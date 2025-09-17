package chi

import (
	"net/http"
	"strings"
	"sync"
)

// Middleware represents a standard HTTP middleware function.
type Middleware func(http.Handler) http.Handler

// Router exposes the subset of chi features used by the ADM backend.
type Router interface {
	http.Handler
	Use(middlewares ...Middleware)
	Route(pattern string, fn func(Router))
	Get(pattern string, handler http.HandlerFunc)
	Post(pattern string, handler http.HandlerFunc)
}

type routeKey struct {
	method string
	path   string
}

type mux struct {
	base        string
	middlewares []Middleware
	routes      map[routeKey]http.Handler
	methods     map[string]map[string]struct{}
	mu          sync.RWMutex
}

// NewRouter creates a new chi-compatible router.
func NewRouter() Router {
	return &mux{
		routes:  make(map[routeKey]http.Handler),
		methods: make(map[string]map[string]struct{}),
	}
}

func (m *mux) Use(middlewares ...Middleware) {
	if len(middlewares) == 0 {
		return
	}
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *mux) Route(pattern string, fn func(Router)) {
	if fn == nil {
		return
	}
	child := &mux{
		base:        joinPath(m.base, pattern),
		middlewares: append([]Middleware{}, m.middlewares...),
		routes:      m.routes,
		methods:     m.methods,
	}
	fn(child)
}

func (m *mux) Get(pattern string, handler http.HandlerFunc) {
	m.handle(http.MethodGet, pattern, handler)
}

func (m *mux) Post(pattern string, handler http.HandlerFunc) {
	m.handle(http.MethodPost, pattern, handler)
}

func (m *mux) handle(method, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		return
	}
	path := joinPath(m.base, pattern)
	wrapped := http.Handler(handler)
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		wrapped = m.middlewares[i](wrapped)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.routes[routeKey{method: method, path: path}] = wrapped
	if _, ok := m.methods[path]; !ok {
		m.methods[path] = make(map[string]struct{})
	}
	m.methods[path][method] = struct{}{}
}

func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := routeKey{method: r.Method, path: cleanPath(r.URL.Path)}

	m.mu.RLock()
	handler, ok := m.routes[key]
	allowed := m.methods[key.path]
	m.mu.RUnlock()

	if ok {
		handler.ServeHTTP(w, r)
		return
	}

	if len(allowed) > 0 {
		allowedMethods := make([]string, 0, len(allowed))
		for method := range allowed {
			allowedMethods = append(allowedMethods, method)
		}
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

func joinPath(base, pattern string) string {
	if pattern == "" {
		pattern = "/"
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if base == "" {
		return cleanPath(pattern)
	}
	if pattern == "/" {
		return cleanPath(base)
	}
	return cleanPath(strings.TrimRight(base, "/") + pattern)
}

func cleanPath(path string) string {
	if path == "" {
		return "/"
	}
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.TrimRight(path, "/")
	}
	return path
}

var _ Router = (*mux)(nil)
