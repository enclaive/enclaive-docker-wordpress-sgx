package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type RestoreInformation struct {
	Host          string `json:"host"`
	BackupDate    string `json:"backup_date"`
	AkeebaVersion string `json:"akeeba_version"`
	PhpVersion    string `json:"php_version"`
	Root          string `json:"root"`
}

type RestoreInformationDatabaseStart struct {
	Percent  int    `json:"percent"`
	Restored int    `json:"restored"`
	Total    string `json:"total"`
	Eta      string `json:"eta"`
	Error    string `json:"error"`
	Done     int    `json:"done"`
}

type RestoreInformationDatabaseStep struct {
	Percent         int    `json:"percent"`
	Restored        string `json:"restored"`
	Total           string `json:"total"`
	QueriesRestored int    `json:"queries_restored"`
	ErrorCount      int    `json:"errorcount"`
	ErrorLog        string `json:"errorlog"`
	CurrentLine     int    `json:"current_line"`
	CurrentPart     int    `json:"current_part"`
	TotalParts      int    `json:"total_parts"`
	Eta             string `json:"eta"`
	Error           string `json:"error"`
	Done            string `json:"done"`
}

func restore(handler http.Handler) {
	installationPath, err := os.Stat(filepath.Join(basePath, "installation"))
	if err != nil || !installationPath.IsDir() {
		log.Println("No backup-installation dir found, skipping restoration")
		return
	}

	log.Println("Starting database restoration")
	var res *http.Response

	var restoreInfoDbStart RestoreInformationDatabaseStart
	res = restoreRequest(handler, restoreStepDatabaseStart())
	check(json.NewDecoder(restoreExpect(res, 200)).Decode(&restoreInfoDbStart))
	log.Printf("Step-Two: %+v\n", restoreInfoDbStart)

	log.Println("Waiting for database restoration")

	var restoreInfoDbStep RestoreInformationDatabaseStep
	for restoreInfoDbStep.Percent != 100 {
		res := restoreRequest(handler, restoreStepDatabaseStep())
		check(json.NewDecoder(restoreExpect(res, 200)).Decode(&restoreInfoDbStep))
		log.Printf("Step-Three: %+v\n", restoreInfoDbStep)
		time.Sleep(time.Second)
	}

	if restoreInfoDbStep.ErrorCount != 0 {
		logFile, err := os.Open(restoreInfoDbStep.ErrorLog)
		if err == nil {
			logFileBytes, err := ioutil.ReadAll(logFile)
			if err == nil {
				log.Println(string(logFileBytes))
			}
		}
		log.Fatalln("Database restoration encountered", restoreInfoDbStep.ErrorCount, "errors")
	}

	log.Println("Replacing site data")

	res = restoreRequest(handler, restoreStepConfig())
	restoreExpect(res, 303)
	log.Println("Step-Four: Done")

	res = restoreRequest(handler, restoreStepReplace())
	restoreExpect(res, 200)
	log.Println("Step-Four-One: Done")

	log.Println("Finalizing restoration")
	var finalizeStatus bool

	res = restoreRequest(handler, restoreStepFinalise())
	check(json.NewDecoder(restoreExpect(res, 200)).Decode(&finalizeStatus))
	log.Printf("Step-Five: %+v\n", finalizeStatus)

	if finalizeStatus {
		log.Fatalln("Finalize should not be ready yet! Aborting.")
	}

	res = restoreRequest(handler, restoreStepUpdateHtaccess())
	check(json.NewDecoder(restoreExpect(res, 200)).Decode(&finalizeStatus))
	log.Printf("Step-Six: %+v\n", finalizeStatus)

	if !finalizeStatus {
		log.Fatalln("Finalize should be done now! Aborting.")
	}

	log.Println("Cleaning up restoration")

	res = restoreRequest(handler, restoreStepCleanup())
	check(json.NewDecoder(restoreExpect(res, 200)).Decode(&finalizeStatus))
	log.Printf("Step-Seven: %+v\n", finalizeStatus)

	if !finalizeStatus {
		log.Fatalln("Cleanup failed! Aborting.")
	}

	log.Println("Successfully restored")
}

func restoreStepDatabaseStart() *url.Values {
	data := &url.Values{}
	data.Add("view", "dbrestore")
	data.Add("task", "start")
	data.Add("format", "json")
	data.Add("key", "site.sql")
	data.Add("dbinfo[dbtype]", "mysqli")
	data.Add("dbinfo[dbhost]", dbHost)
	data.Add("dbinfo[dbuser]", dbUser)
	data.Add("dbinfo[dbpass]", dbPass)
	data.Add("dbinfo[dbname]", dbName)
	data.Add("dbinfo[prefix]", dbPrefix)
	data.Add("dbinfo[existing]", "drop")
	data.Add("dbinfo[foreignkey]", "1")
	data.Add("dbinfo[noautovalue]", "1")
	data.Add("dbinfo[replace]", "0")
	data.Add("dbinfo[utf8db]", "0")
	data.Add("dbinfo[utf8tables]", "0")
	data.Add("dbinfo[utf8mb4]", "1")
	data.Add("dbinfo[break_on_failed_create]", "1")
	data.Add("dbinfo[break_on_failed_insert]", "1")
	data.Add("dbinfo[maxexectime]", "5")
	data.Add("dbinfo[throttle]", "250")
	return data
}

func restoreStepDatabaseStep() *url.Values {
	data := &url.Values{}
	data.Set("view", "dbrestore")
	data.Set("task", "step")
	data.Set("format", "json")
	data.Set("key", "site.sql")
	return data
}

func restoreStepConfig() *url.Values {
	data := &url.Values{}
	data.Set("view", "setup")
	data.Set("task", "apply")
	data.Set("homeurl", "https://"+hostName)
	data.Set("siteurl", "https://"+hostName)
	return data
}

func restoreStepReplace() *url.Values {
	data := &url.Values{}
	data.Set("view", "replacedata")
	data.Set("task", "ajax")
	data.Set("method", "init")
	data.Set("format", "json")

	oldHost := restoreInfoHostname()
	oldNames := []string{"https://" + oldHost, "https:\\/\\/" + oldHost}
	newNames := []string{"https://" + hostName, "https:\\/\\/" + hostName}
	data.Set("replaceFrom", strings.Join(oldNames, "\u0000"))
	data.Set("replaceTo", strings.Join(newNames, "\u0000"))

	// db settings
	data.Set("runtime_bias", "75")
	data.Set("max_exec", "3")
	data.Set("min_exec", "0")
	data.Set("batchSize", "100")

	// has impact on rss readers
	data.Set("replaceguid", "1")

	// tables to include in replace
	extraTables := []string{
		"wp_ak_params",
		"wp_ak_profiles",
		"wp_ak_stats",
		"wp_ak_storage",
		"wp_ak_users",
		"wp_akeeba_common",
		"wp_termmeta",
	}

	data.Del("extraTables[]")
	for _, extraTable := range extraTables {
		data.Add("extraTables[]", extraTable)
	}

	return data
}

func restoreStepFinalise() *url.Values {
	data := &url.Values{}
	data.Set("view", "finalise")
	data.Set("task", "ajax")
	data.Set("format", "json")
	return data
}

func restoreStepUpdateHtaccess() *url.Values {
	data := restoreStepFinalise()
	data.Set("method", "updatehtaccess")
	return data
}

func restoreStepCleanup() *url.Values {
	data := restoreStepFinalise()
	data.Set("task", "cleanup")
	return data
}

func restoreExpect(res *http.Response, statusCode int) io.Reader {
	if res.StatusCode != statusCode {
		log.Fatalln("Unexpected status code while restoring:", res.StatusCode, "on:", res.Request.URL.String())
	}

	resBytes, err := io.ReadAll(res.Body)
	check(err)

	return strings.NewReader(strings.Trim(string(resBytes), "#"))
}

func restoreRequest(handler http.Handler, data *url.Values) *http.Response {
	data.Add("_dontcachethis", strconv.FormatFloat(rand.Float64(), 'E', 3, 64))

	restoreUrl := fmt.Sprintf("https://%s/installation/index.php", hostName)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, restoreUrl, strings.NewReader(data.Encode()))

	request.Header.Set("x-requested-with", "XMLHttpRequest")
	request.Header.Set("user-agent", "restore/internal")
	request.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")

	log.Println(request.URL.String())
	for name, header := range request.Header {
		log.Printf("%s: %+v\n", name, header)
	}

	handler.ServeHTTP(recorder, request)

	return recorder.Result()
}

func restoreInfoHostname() string {
	restoreInfoPath := filepath.Join(basePath, "installation", "extrainfo.json")
	restoreInfoStat, err := os.Stat(restoreInfoPath)
	check(err)

	if restoreInfoStat.IsDir() {
		log.Fatalln("extrainfo.json should not be a directory")
	}

	restoreInfoFile, err := os.Open(restoreInfoPath)
	check(err)

	var restoreInfo RestoreInformation
	check(json.NewDecoder(restoreInfoFile).Decode(&restoreInfo))

	return restoreInfo.Host
}
