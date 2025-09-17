package server

import (
	"net/http"
	"strings"
	"time"

	"adm-backend/internal/api"
)

// NewMux assembles the HTTP handlers for the ADM backend.
func NewMux(adminHandler *api.AdminHandler, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})

	api.RegisterStudentRoutes(mux)
	api.RegisterAdminRoutes(mux, adminHandler)

	return withCORS(mux, allowedOrigins)
}

// NewHTTPServer returns an http.Server instance configured with sensible timeouts.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func withCORS(next http.Handler, origins []string) http.Handler {
	allowed := make(map[string]struct{})
	allowAll := false
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
		}
		allowed[trimmed] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowAll || hasOrigin(allowed, origin)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		} else if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hasOrigin(set map[string]struct{}, origin string) bool {
	if _, ok := set[origin]; ok {
		return true
	}

	// Handle variations like trailing slash removal.
	trimmed := strings.TrimRight(origin, "/")
	if trimmed != origin {
		if _, ok := set[trimmed]; ok {
			return true
		}
	}

	// Support local loopback equivalence between localhost and 127.0.0.1
	if strings.HasPrefix(origin, "http://127.0.0.1") {
		candidate := strings.Replace(origin, "127.0.0.1", "localhost", 1)
		if _, ok := set[candidate]; ok {
			return true
		}
	}
	if strings.HasPrefix(origin, "http://localhost") {
		candidate := strings.Replace(origin, "localhost", "127.0.0.1", 1)
		if _, ok := set[candidate]; ok {
			return true
		}
	}

	return false
}
