package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
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
	Percent  int         `json:"percent"`
	Restored int         `json:"restored"`
	Total    interface{} `json:"total"`
	Eta      interface{} `json:"eta"`
	Error    interface{} `json:"error"`
	Done     interface{} `json:"done"`
}

type RestoreInformationDatabaseStep struct {
	Percent         int         `json:"percent"`
	Restored        interface{} `json:"restored"`
	Total           interface{} `json:"total"`
	Eta             interface{} `json:"eta"`
	Error           interface{} `json:"error"`
	Done            interface{} `json:"done"`
	QueriesRestored int         `json:"queries_restored"`
	ErrorCount      int         `json:"errorcount"`
	ErrorLog        string      `json:"errorlog"`
	CurrentLine     int         `json:"current_line"`
	CurrentPart     int         `json:"current_part"`
	TotalParts      int         `json:"total_parts"`
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

	newLocation, err := res.Location()
	check(err)

	// tables to include in replace
	extraTables := restoreExtraTables(handler, newLocation.String())

	if len(extraTables) == 0 {
		log.Fatalln("Encountered an empty extraTables extraction")
	}

	res = restoreRequest(handler, restoreStepReplace(extraTables))
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

func restoreExtraTables(handler http.Handler, location string) []string {
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, location, strings.NewReader(""))
	check(err)

	request.Header.Set("User-Agent", "restore/internal")

	handler.ServeHTTP(recorder, request)

	res := recorder.Result()

	tok := html.NewTokenizer(restoreExpect(res, 200))

	var out []string
	for {
		token := tok.Next()

		if token == html.ErrorToken {
			break
		}

		tagName, hasAttr := tok.TagName()

		if token == html.StartTagToken && string(tagName) == "select" && hasAttr {
			var key, value []byte
			for hasAttr {
				key, value, hasAttr = tok.TagAttr()
				if string(key) == "id" && string(value) == "extraTables" {
					out = restoreExtraTablesExtract(tok)
				}
			}
		}
	}
	return out
}

func restoreExtraTablesExtract(tok *html.Tokenizer) []string {
	var out []string

	for {
		token := tok.Next()

		if token == html.ErrorToken {
			break
		}

		tagName, hasAttr := tok.TagName()

		if token == html.StartTagToken {
			if string(tagName) == "option" && hasAttr {
				var key, value []byte
				for hasAttr {
					key, value, hasAttr = tok.TagAttr()
					if string(key) == "value" {
						out = append(out, string(value))
					}
				}
			} else if string(tagName) == "select" {
				log.Fatalln("nested select")
			}
		} else if token == html.EndTagToken {
			if string(tagName) == "select" {
				break
			}
		}
	}

	return out
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

func restoreStepReplace(extraTables []string) *url.Values {
	data := &url.Values{}
	data.Set("view", "replacedata")
	data.Set("task", "ajax")
	data.Set("method", "init")
	data.Set("format", "json")

	oldInfo := restoreInformation()
	oldHost := oldInfo.Host
	oldRoot := oldInfo.Root
	oldRootEsc := strings.Join(strings.Split(oldRoot, "/"), "\\/")
	basePathEsc := strings.Join(strings.Split(basePath, "/"), "\\/")

	//goland:noinspection HttpUrlsUsage
	oldNames := []string{
		oldRoot,
		oldRootEsc,
		"http://" + oldHost,
		"http:\\/\\/" + oldHost,
		"https://" + oldHost,
		"https:\\/\\/" + oldHost,
	}
	newNames := []string{
		basePath,
		basePathEsc,
		"https://" + hostName,
		"https:\\/\\/" + hostName,
		"https://" + hostName,
		"https:\\/\\/" + hostName,
	}

	log.Println("Replacing from oldNames:", oldNames)
	log.Println("Replacing to newNames:", newNames)

	data.Set("replaceFrom", strings.Join(oldNames, "\u0000"))
	data.Set("replaceTo", strings.Join(newNames, "\u0000"))

	// db settings
	data.Set("runtime_bias", "75")
	data.Set("max_exec", "3")
	data.Set("min_exec", "0")
	data.Set("batchSize", "100")

	// has impact on rss readers
	data.Set("replaceguid", "1")

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
	request, err := http.NewRequest(http.MethodPost, restoreUrl, strings.NewReader(data.Encode()))
	check(err)

	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", "restore/internal")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	handler.ServeHTTP(recorder, request)

	return recorder.Result()
}

func restoreInformation() RestoreInformation {
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

	return restoreInfo
}
