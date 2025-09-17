package server

import (
	"net/http"
	"strings"
	"time"

	"adm-backend/internal/api"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter assembles the HTTP handlers for the ADM backend using chi.
func NewRouter(adminHandler *api.AdminHandler, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	cleanOrigins := sanitizeOrigins(allowedOrigins)
	corsOptions := cors.Options{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-User-Login"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}

	if len(cleanOrigins) == 0 {
		corsOptions.AllowedOrigins = []string{"http://localhost:8080", "http://localhost:8081"}
	} else if containsWildcard(cleanOrigins) {
		corsOptions.AllowOriginFunc = func(_ *http.Request, _ string) bool { return true }
	} else {
		corsOptions.AllowedOrigins = cleanOrigins
	}

	r.Use(cors.Handler(corsOptions))

	registerAPIRoutes := func(router chi.Router) {
		router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		})

		router.Route("/student", func(sr chi.Router) {
			api.RegisterStudentRoutes(sr)
		})

		router.Route("/admin", func(ar chi.Router) {
			api.RegisterAdminRoutes(ar, adminHandler)
		})
	}

	registerAPIRoutes(r)

	r.Route("/api", func(ar chi.Router) {
		registerAPIRoutes(ar)
	})

	return r
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

func sanitizeOrigins(origins []string) []string {
	cleaned := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func containsWildcard(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}
	return false
}

// WithBasePath optionally strips a configured base path before forwarding the request.
func WithBasePath(next http.Handler, basePath string) http.Handler {
	clean := sanitizeBasePath(basePath)
	if clean == "" || clean == "/" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, clean) {
			http.NotFound(w, r)
			return
		}

		trimmed := path[len(clean):]
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}

		clone := r.Clone(r.Context())
		clone.URL.Path = trimmed
		clone.URL.RawPath = trimmed
		next.ServeHTTP(w, clone)
	})
}

func sanitizeBasePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if trimmed != "/" {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}
