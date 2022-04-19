package main

// #include "bridge.h"
// #include <stdlib.h>
import "C"

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "net/http"
    "strings"
    "os"
    "path/filepath"
    "sync"
    "github.com/mattn/go-pointer"
    "unsafe"
)

type Context struct {
    w               http.ResponseWriter
    r               *http.Request
    scriptPath      string
    dtor            []func()
}

// php isnt thread safe yet?
var BIGLOCK sync.Mutex

func handlePHP(absroot string, path string, w http.ResponseWriter, r *http.Request) {

    BIGLOCK.Lock()
    defer BIGLOCK.Unlock();

    var ctx = &Context{
        w: w,
        r: r,
        scriptPath: "." + path,
    };

    defer func() {
        log.Println("cleanup");
        for _, dtor := range ctx.dtor {
            dtor();
        }
    }()

    var c = pointer.Save(ctx);
    defer pointer.Unref(c);
    log.Printf("SAVE CTX PTR %x", c);


    //safety
    strings.ReplaceAll(path, "..", "")
    if strings.HasPrefix(path, "/") {
        path = "." + path
    }

    var script_path = C.CString(filepath.Join(absroot, path))
    defer C.free(unsafe.Pointer(script_path))

    var request_method = C.CString(r.Method);
    defer C.free(unsafe.Pointer(request_method));

    var request_uri = C.CString(r.RequestURI);
    defer C.free(unsafe.Pointer(request_uri));

    var query_string = C.CString(r.URL.RawQuery);
    defer C.free(unsafe.Pointer(query_string));

    var content_type = C.CString(r.Header.Get("Content-Type"));
    defer C.free(unsafe.Pointer(content_type));

    var content_length = C.size_t(r.ContentLength);

    C.phpmain(
        c,
        script_path,
        request_method,
        request_uri,
        query_string,
        content_type,
        content_length,
    );
    log.Println("phpmain should be done");
}

func main() {

    fmt.Println("starting");

    os.Chdir("wordpress")
    root, err := filepath.Abs(".");
    if err != nil { panic(err) }


    fs := http.FileServer(http.Dir(root))

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

        log.Println(r.URL);

        if strings.HasSuffix(r.URL.Path, "/") {
            r.URL.Path += "index.php"
        }
        if strings.HasSuffix(r.URL.Path, ".php") {
            handlePHP(root, r.URL.Path, w, r);
        } else {
            fs.ServeHTTP(w, r);
        }
    })

    fmt.Println("listening on 0.0.0.0:3000");
    panic(http.ListenAndServe("127.0.0.1:3000", nil))
}
