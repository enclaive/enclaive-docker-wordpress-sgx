package main

// #include "bridge.h"
// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"
	"github.com/mattn/go-pointer"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"
    "io"
)

var (
	hostName = os.Getenv("HOST_NAME")
)

const (
	basePath = "/app/wordpress"
	dbHost   = "edb"
	dbName   = "wordpress"
	dbPrefix = "wp_"
	dbUser   = "root"
	dbPass   = "root"
)

type Context struct {
	w              http.ResponseWriter
	r              *http.Request
	phpSelf        string
	scriptName     string
	scriptFilename string
	dtor           []func()
	done           chan struct{}
}

var PHPW = make(chan Context, 5)

func phpW() {
	runtime.LockOSThread()

	for ctx := range PHPW {
		phpOnce(&ctx)
	}
}

func phpOnce(ctx *Context) {
	defer func() {

		if err := recover(); err != nil {
			ctx.w.WriteHeader(500)
			if err, ok := err.(error); ok {
				ctx.w.Write([]byte(err.Error()))
			}
		}

		ctx.done <- struct{}{}
		log.Println("cleanup")
		for _, dtor := range ctx.dtor {
			dtor()
		}
	}()

	err := os.Chdir(basePath)
	if err != nil {
		panic(err)
	}

	var c = pointer.Save(ctx)
	defer pointer.Unref(c)
	log.Printf("SAVE CTX PTR %x", c)

	pathScript, err := scriptPath(ctx.r.URL.Path)

	if err != nil {
		log.Println("phpmain was illegally called, this should not have happened")
		ctx.w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.phpSelf = strings.TrimPrefix(pathScript, basePath)
	ctx.scriptFilename = pathScript
	ctx.scriptName = ctx.phpSelf

	var script_path = C.CString(pathScript)
	defer C.free(unsafe.Pointer(script_path))

	var request_method = C.CString(ctx.r.Method)
	defer C.free(unsafe.Pointer(request_method))

	var request_uri = C.CString(ctx.r.RequestURI)
	defer C.free(unsafe.Pointer(request_uri))

	var query_string = C.CString(ctx.r.URL.RawQuery)
	defer C.free(unsafe.Pointer(query_string))

	var content_type = C.CString(ctx.r.Header.Get("Content-Type"))
	defer C.free(unsafe.Pointer(content_type))

	var content_length = C.size_t(ctx.r.ContentLength)

	C.phpmain(
		c,
		script_path,
		request_method,
		request_uri,
		query_string,
		content_type,
		content_length,
	)
	log.Println("phpmain should be done")
}

func scriptPath(urlPath string) (string, error) {
	joined := filepath.Join(basePath, path.Clean(urlPath))
	evaluated, err := filepath.EvalSymlinks(joined)

	if err != nil {
		// ignore missing files, e.g. virtual urls resolved by WordPress
		if _, ok := err.(*os.PathError); !ok {
			return "", err
		} else {
			evaluated = joined
		}
	}

	cleaned := path.Clean(evaluated)

	if !path.IsAbs(cleaned) {
		return "", errors.New("unexpected non-absolute path encountered: " + cleaned)
	}

	if !strings.HasPrefix(cleaned, basePath+"/") && cleaned != basePath {
		return "", errors.New("the script path is outside of webroot: " + cleaned)
	}

	// WordPress is doing crazy stuff, guessing the url based on PHP_SELF,
	// so we should use relative script paths to avoid leaking information,
	// but this breaks so much other stuff, wow...
	return cleaned, nil
}

func main() {
	fmt.Println("starting")

    GramineSetup()

    /*
	ExtractAppZip()

	//TODO spawn more workers. but php needs thread locals and i'm not confident yet they actually work correctly in gramine.
	go phpW()

    */


	fs := http.FileServer(http.Dir(basePath))

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("")
		fmt.Println("")
		fmt.Println("----------------------------------------------------------------")
		log.Println(r.URL)

        if r.URL.Path == "/dev/attestation/report" {
            f, err := os.Open("/dev/attestation/report")
            if err != nil { panic(err) }
            defer f.Close();

            io.Copy(w, f)
            return;
        } else if r.URL.Path == "/dev/attestation/quote" {
            f, err := os.Open("/dev/attestation/quote")
            if err != nil { panic(err) }
            defer f.Close();

            io.Copy(w, f)
            return;
        }

		requestPath, err := scriptPath(r.URL.Path)



		if err != nil {
			log.Println("illegal request to", requestPath, "resulted in", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fileInfo, err := os.Stat(requestPath)

		if err == nil {
			if fileInfo.IsDir() {
				if strings.HasSuffix(r.URL.Path, "/") {
					r.URL.Path = path.Clean(filepath.Join(r.URL.Path, "index.php"))
				} else {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
				}
			}
		} else {
			r.URL.Path = "/index.php"
		}

		if strings.HasSuffix(r.URL.Path, ".php") {
			var ctx = Context{
				w:    w,
				r:    r,
				done: make(chan struct{}),
			}

			select {
			case PHPW <- ctx:
				<-ctx.done
			case <-time.After(10 * time.Second):
				close(ctx.done)
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("all enclaves busy. try again later"))
			}

			log.Println("phpmain should be done")

		} else {
			fs.ServeHTTP(w, r)
		}
	})

	handler := tracing(NewApacheLoggingHandler(logging(router)))
	//handler := tracing(logging(caching(router)))
	//go cachingRequest()

	fmt.Println("Trying to restore from a backup")
	restore(handler)

	fmt.Println("listening on https://0.0.0.0:443")
	panic(http.ListenAndServeTLS("0.0.0.0:443", "/app/tls/server.crt", "/app/tls/server.key", handler))
}
