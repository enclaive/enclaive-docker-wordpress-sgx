package main


// #include "bridge.h"
// #include <stdlib.h>
// #include <string.h>
// #cgo CFLAGS: -fpic -I/usr/include/php/ -I/usr/include/php/Zend -I/usr/include/php/main/ -I/usr/include/php/TSRM
// #cgo LDFLAGS: -lphp  -llzma -lxml2 -lz -lm -lpthread -lcrypto -lssl -lsqlite3 -lpng -lzip -lbz2 -largon2 -lreadline -lcurl
import "C"

import (
    "unsafe"
    "fmt"
    log "github.com/sirupsen/logrus"
    "strings"
    "github.com/mattn/go-pointer"
)


// backport for go1.17
func strings_Cut(s, sep string) (before, after string, found bool) {
    if i := strings.Index(s, sep); i >= 0 {
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
    gophp_register_variables_each_go(ctx, p, "HTTP_HOST",       ctx.r.Host);
    gophp_register_variables_each_go(ctx, p, "SERVER_NAME",     ctx.r.Host);
    gophp_register_variables_each_go(ctx, p, "REMOTE_ADDR",     ctx.r.RemoteAddr);
    gophp_register_variables_each_go(ctx, p, "HTTPS",           "on");

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
