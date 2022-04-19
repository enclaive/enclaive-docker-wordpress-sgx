package main



// #cgo CFLAGS: -I./php-8.1.4/ -I./php-8.1.4/Zend -I./php-8.1.4/main -I./php-8.1.4/TSRM
// #include "sapi/embed/php_embed.h"
// #include "gophp.h"
// #cgo LDFLAGS: php-8.1.4/libs/libphp.a -l:libxml2.a -l:liblzma.a -l:libicuuc.a -l:libicudata.a -l:libstdc++.a
// #cgo LDFLAGS: -l:libcurl.a -l:libssh.a -l:libldap.a -l:libgss.a -l:librtmp.a -l:libgnutls.a -l:libhogweed.a -l:libnettle.a -l:libtasn1.a -l:libgcrypt.a -l:libgpg-error.a -l:libidn2.a -l:libunistring.a -l:libgmp.a -l:liblber.a -l:libpsl.a -l:libnghttp2.a -l:libunistring.a -l:libbrotlidec.a -l:libbrotlicommon.a
// #cgo LDFLAGS: -lp11-kit -lshishi -lsasl2
// #cgo LDFLAGS: -l:libsqlite3.a -l:libpng.a -l:libargon2.a
// #cgo LDFLAGS: -l:libbz2.a -l:libcrypto.a -l:libz.a -l:libssl.a -l:libcrypto.a
// #cgo LDFLAGS: -l:libreadline.a -l:libtinfo.a
// #cgo LDFLAGS: -lzip
// #cgo LDFLAGS: -lc-client
// #cgo LDFLAGS: -lm
import "C"

import (
    "unsafe"
    "fmt"
    log "github.com/sirupsen/logrus"
    "net/http"
    "github.com/mattn/go-pointer"
    "strings"
    "os"
    "path/filepath"
    "sync"
)

// backport for go1.17
func Cut(s, sep string) (before, after string, found bool) {
    if i := Index(s, sep); i >= 0 {
        return s[:i], s[i+len(sep):], true
    }
    return s, "", false
}

//export gophp_body_write
func gophp_body_write(r *C.void, m *C.char, l C.size_t) C.size_t {
    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    l2, _ := ctx.w.Write(C.GoBytes(unsafe.Pointer(m), C.int(l)));
    return C.size_t(l2);
}

//export gophp_response_headers_write
func gophp_response_headers_write (r *C.void, response_code C.int) {
    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    fmt.Println("gophp_response_headers_write!", response_code)
    ctx.w.WriteHeader(int(response_code));
}
//export gophp_response_headers_add
func gophp_response_headers_add   (r *C.void, m *C.char, l C.size_t) {


    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    k,v,ok := strings_Cut(C.GoStringN(m, C.int(l)), ":");
    fmt.Println("gophp_response_headers_add", k, v)
    if ok {
        ctx.w.Header().Add(k,v)
    }
}
//export gophp_response_headers_del
func gophp_response_headers_del   (r *C.void, m *C.char, l C.size_t) {
    fmt.Println("gophp_response_headers_del")

    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    ctx.w.Header().Del(C.GoStringN(m, C.int(l)))
}
//export gophp_response_headers_set
func gophp_response_headers_set   (r *C.void, m *C.char, l C.size_t) {
    fmt.Println("gophp_response_headers_set", C.GoStringN(m, C.int(l)))

    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    k,v,ok := strings_Cut(C.GoStringN(m, C.int(l)), ":");
    if ok {
        ctx.w.Header().Set(k,v)
    }
}
//export gophp_response_headers_clear
func gophp_response_headers_clear (r *C.void) {
    fmt.Println("gophp_response_headers_clear")

    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    for k,_ := range ctx.w.Header() {
        ctx.w.Header().Del(k);
    }
}




func gophp_register_variables_each_go(ctx *Context, p *C.void, k string, v string) {
    cstrK := C.CString(k);
    cstrV := C.CString(v)
    ctx.dtor = append(ctx.dtor, func() {
        C.free(unsafe.Pointer(cstrK));
        C.free(unsafe.Pointer(cstrV));
    });
    C.gophp_register_variables_each_php(unsafe.Pointer(p), cstrK, cstrV);
}

//export gophp_register_variables_go
func gophp_register_variables_go(r *C.void, p *C.void) {

    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)

    gophp_register_variables_each_go(ctx, p, "REQUEST_METHOD",  ctx.r.Method);
    gophp_register_variables_each_go(ctx, p, "REQUEST_URI",     ctx.r.RequestURI);
    gophp_register_variables_each_go(ctx, p, "PHP_SELF",        ctx.scriptPath);
    gophp_register_variables_each_go(ctx, p, "SCRIPT_FILENAME", ctx.scriptPath);
    gophp_register_variables_each_go(ctx, p, "SCRIPT_NAME",     ctx.scriptPath);
    gophp_register_variables_each_go(ctx, p, "HTTP_HOST",       "localhost:3000");
    gophp_register_variables_each_go(ctx, p, "SERVER_NAME",     "localhost:3000");
    gophp_register_variables_each_go(ctx, p, "REMOTE_ADDR",     "127.0.0.1");

    for k,vv := range ctx.r.Header {
        gophp_register_variables_each_go(ctx, p, "HTTP_" + strings.ReplaceAll(strings.ToUpper(k), "-", "_"), vv[0]);
    }

}

//export gophp_request_get_cookie
func gophp_request_get_cookie(r *C.void) *C.char{
    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)

    var ck = ctx.r.Header.Get("Cookie");
    log.Println("Cookie", ck);

    cstr := C.CString(ck)
    ctx.dtor = append(ctx.dtor, func() {
        C.free(unsafe.Pointer(cstr));
    })
    return cstr;
}

//export gophp_request_read_post
func gophp_request_read_post(r *C.void, buf *C.char, l C.size_t) C.size_t {

    ctx := pointer.Restore(unsafe.Pointer(r)).(*Context)
    var b = make([]byte, int(l));
    l2, _ := ctx.r.Body.Read(b)
    C.memcpy(unsafe.Pointer(buf), unsafe.Pointer(&b[0]), C.size_t(l2));
    return C.size_t(l2);
}

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

    fmt.Println("listening on :3000");
    http.ListenAndServe(":3000", nil)
}
