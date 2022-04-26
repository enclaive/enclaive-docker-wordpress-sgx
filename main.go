package main

// #include "bridge.h"
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"github.com/mattn/go-pointer"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

type Context struct {
	w          http.ResponseWriter
	r          *http.Request
	scriptPath string
	dtor       []func()
	done       chan struct{}
}

var PHPW = make(chan Context, 1)

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

	err := os.Chdir("/app/wordpress")
	if err != nil {
		panic(err)
	}

	var c = pointer.Save(ctx)
	defer pointer.Unref(c)
	log.Printf("SAVE CTX PTR %x", c)

	ctx.scriptPath = "." + ctx.r.URL.Path
	//TODO more safety? since the VFS is a kernel emulation that allows escaping the mount
	strings.ReplaceAll(ctx.scriptPath, "..", "")
	if strings.HasPrefix(ctx.scriptPath, "/") {
		ctx.scriptPath = "." + ctx.scriptPath
	}

	var script_path = C.CString(filepath.Join("/app/wordpress/", ctx.scriptPath))
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

func main() {

	fmt.Println("starting")

	ExtractAppZip()

	//TODO spawn more workers. but php needs thread locals and i'm not confident yet they actually work correctly in gramine.
	go phpW()

	fs := http.FileServer(http.Dir("/app/wordpress"))

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Println(r.URL)

		if strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path += "index.php"
		}
		if strings.HasSuffix(r.URL.Path, ".php") {

			//safety
			strings.ReplaceAll(r.URL.Path, "..", "")

			var ctx = Context{
				w:    w,
				r:    r,
				done: make(chan struct{}),
			}

			select {
			case PHPW <- ctx:
				<-ctx.done
			case <-time.After(time.Second):
				close(ctx.done)
				w.WriteHeader(503)
				w.Write([]byte("all enclaves busy. try again later"))
			}

			log.Println("phpmain should be done")

		} else {
			fs.ServeHTTP(w, r)
		}
	})

	handler := tracing(logging(caching(router)))
	go cachingRequest()

	fmt.Println("listening on https://0.0.0.0:443")
	panic(http.ListenAndServeTLS("0.0.0.0:443", "/app/server.crt", "/app/server.key", handler))
}
