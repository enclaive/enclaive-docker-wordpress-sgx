package main

import (
	"context"
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

type requestIDKeyType int

var requestIDKey requestIDKeyType = 0

var requestCachingPaths = map[string]bool{
	"/": true,
}

var requestCachingStore = map[string]requestCacheResponse{}
var requestCachingStoreSync = &sync.Mutex{}

const requestCachingTTL = 5 * time.Minute
const requestCachingInterval = 1 * time.Minute

type requestCacheResponse struct {
	status  int
	header  http.Header
	body    []byte
	expires time.Time
}

// FIXME: use http.Request instead of networking through the host. Currently that method would create a segfault,
// 	because some parameters are not set appropriately
func cachingRequest() {
	ticker := time.NewTicker(requestCachingInterval)
	defer ticker.Stop()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	action := func() {
		for path := range requestCachingPaths {
			response, err := client.Get("https://127.0.0.1:443" + path)

			if err != nil {
				log.Println("[cache] Interval request for", path, "failed with", err)
				continue
			}

			log.Println("[cache] Interval request response:", response.StatusCode)
		}
	}

	action()

	for range ticker.C {
		action()
	}
}

// TODO: prevent read race conditions on map requestCachingStore
// TODO: optionally, implement sync locks for concurrent requests on the same path
// adapted from: https://github.com/victorspringer/http-cache/blob/master/cache.go
func caching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond := func(status int, header http.Header, body []byte) {
			for k, v := range header {
				w.Header().Set(k, strings.Join(v, ","))
			}

			w.WriteHeader(status)
			_, err := w.Write(body)
			if err != nil {
				log.Println("[cache] Error writing response", err)
				return
			}
		}

		// only handle get without query parameters for now
		// exclude http-auth just in case
		if r.Method == http.MethodGet && len(r.URL.Query()) == 0 && r.URL.User.String() == "" {
			path := r.URL.Path

			log.Println("[cache] Received GET request for", path)

			if stored, found := requestCachingStore[path]; found && stored.expires.After(time.Now()) {
				log.Println("[cache] Path found in cache-store and response is not expired")

				respond(stored.status, stored.header, stored.body)
			} else if _, found = requestCachingPaths[path]; found {
				log.Println("[cache] Path is marked for caching")

				recorder := httptest.NewRecorder()

				next.ServeHTTP(recorder, r)

				result := recorder.Result()
				response := recorder.Body.Bytes()

				if result.StatusCode == 200 {
					log.Println("[cache] Request succeeded with status code", result.StatusCode, "adding to cache")

					requestCachingStoreSync.Lock()
					requestCachingStore[path] = requestCacheResponse{
						status:  result.StatusCode,
						body:    response,
						header:  result.Header,
						expires: time.Now().Add(requestCachingTTL),
					}
					requestCachingStoreSync.Unlock()
				} else {
					log.Println("[cache] Request failed with status code", result.StatusCode, "not caching")
				}

				respond(result.StatusCode, result.Header, response)
			} else {
				log.Println("[cache] Path is not marked for caching")

				next.ServeHTTP(w, r)
			}
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			requestID, ok := r.Context().Value(requestIDKey).(string)
			if !ok {
				requestID = "unknown"
			}
			fmt.Println("[HTTPD] FIN ", requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		}()
		next.ServeHTTP(w, r)
	})
}
