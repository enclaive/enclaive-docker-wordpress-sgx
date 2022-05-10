package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// https://httpd.apache.org/docs/2.2/logs.html#combined + execution time.
const apacheFormatPattern = "%s - - [%s] \"%s %s %s\" %d %d \"%s\" \"%s\" %.4f\n"

var apacheLog = make(chan string)

type ApacheLogRecord struct {
	http.ResponseWriter

	ip                    string
	time                  time.Time
	method, uri, protocol string
	status                int
	responseBytes         int64
	referer               string
	userAgent             string
	elapsedTime           time.Duration
}

func (r *ApacheLogRecord) Log() string {
	timeFormatted := r.time.Format("02/Jan/2006 03:04:05")
	return fmt.Sprintf(apacheFormatPattern, r.ip, timeFormatted, r.method,
		r.uri, r.protocol, r.status, r.responseBytes, r.referer, r.userAgent,
		r.elapsedTime.Seconds())
}

func (r *ApacheLogRecord) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.responseBytes += int64(written)
	return written, err
}

func (r *ApacheLogRecord) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type ApacheLoggingHandler struct {
	handler http.Handler
}

func NewApacheLoggingHandler(handler http.Handler) http.Handler {
	accessLogFile, err := os.OpenFile("/data/access.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(fmt.Errorf("Error creating /data/access.log : %w ", err))
	}

	go func() {
		defer accessLogFile.Close()

		_, err := accessLogFile.Write([]byte("Access log opened at " + time.Now().UTC().String()))
		check(err)

		check(accessLogFile.Sync())

		for msg := range apacheLog {
			_, err := accessLogFile.Write([]byte(msg))
			check(err)

			check(accessLogFile.Sync())
		}
	}()

	return &ApacheLoggingHandler{handler}
}

func (h *ApacheLoggingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
		clientIP = clientIP[:colon]
	}

	referer := r.Referer()
	if referer == "" {
		referer = "-"
	}

	userAgent := r.UserAgent()
	if userAgent == "" {
		userAgent = "-"
	}

	record := &ApacheLogRecord{
		ResponseWriter: rw,
		ip:             clientIP,
		time:           time.Time{},
		method:         r.Method,
		uri:            r.RequestURI,
		protocol:       r.Proto,
		status:         http.StatusOK,
		referer:        referer,
		userAgent:      userAgent,
		elapsedTime:    time.Duration(0),
	}

	startTime := time.Now()
	h.handler.ServeHTTP(record, r)
	finishTime := time.Now()

	record.time = finishTime.UTC()
	record.elapsedTime = finishTime.Sub(startTime)

	apacheLog <- record.Log()
}
