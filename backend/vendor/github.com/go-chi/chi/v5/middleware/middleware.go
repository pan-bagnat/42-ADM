package middleware

import (
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// RequestID sets a simple incremental request id header.
func RequestID(next http.Handler) http.Handler {
	var counter uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := atomic.AddUint64(&counter, 1)
		w.Header().Set("X-Request-ID", formatID(id))
		next.ServeHTTP(w, r)
	})
}

func formatID(id uint64) string {
	return "req-" + strconvFormat(id)
}

func strconvFormat(v uint64) string {
	const base = 36
	buf := make([]byte, 0, 16)
	for v >= base {
		buf = append([]byte{digits[v%base]}, buf...)
		v /= base
	}
	buf = append([]byte{digits[v]}, buf...)
	return string(buf)
}

var digits = []byte("0123456789abcdefghijklmnopqrstuvwxyz")

// RealIP sets X-Real-IP based on the request remote address when missing.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Real-IP") == "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				r.Header.Set("X-Real-IP", host)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Logger outputs a simple access log line.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// Recoverer captures panics and responds with 500.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
