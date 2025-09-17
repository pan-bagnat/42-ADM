package cors

import (
	"net/http"
	"strconv"
	"strings"
)

// Options configures the CORS middleware.
type Options struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
	AllowOriginFunc  func(r *http.Request, origin string) bool
}

// Handler returns a middleware that applies the provided CORS options.
func Handler(options Options) func(http.Handler) http.Handler {
	allowedMethods := strings.Join(deduplicate(options.AllowedMethods), ", ")
	allowedHeaders := strings.Join(deduplicate(options.AllowedHeaders), ", ")
	exposedHeaders := strings.Join(deduplicate(options.ExposedHeaders), ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !isOriginAllowed(r, origin, options) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			if exposedHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			}
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if allowedMethods != "" {
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			}
			if allowedHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			} else {
				if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
				}
			}
			if options.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(options.MaxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(r *http.Request, origin string, options Options) bool {
	if options.AllowOriginFunc != nil {
		return options.AllowOriginFunc(r, origin)
	}
	allowed := deduplicate(options.AllowedOrigins)
	if len(allowed) == 0 {
		return false
	}
	for _, value := range allowed {
		if value == "*" || value == origin {
			return true
		}
	}
	return false
}

func deduplicate(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
