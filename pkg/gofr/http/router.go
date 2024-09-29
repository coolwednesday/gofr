package http

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Router is responsible for routing HTTP request.
type Router struct {
	*httprouter.Router
	RegisteredRoutes *[]string
	middlewares      []Middleware
}

type Middleware func(handler http.Handler) http.Handler

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	httpRouter := httprouter.New()
	httpRouter.RedirectTrailingSlash = false
	routes := make([]string, 0)
	r := &Router{
		Router:           httpRouter,
		RegisteredRoutes: &routes,
		middlewares:      []Middleware{},
	}

	r.Router = httpRouter

	return r
}

// Add adds a new route with the given HTTP method, pattern, and handler, wrapping the handler with OpenTelemetry instrumentation.
func (rou *Router) Add(method, pattern string, handler http.Handler) {
	// Wrap the handler with all middlewares
	for _, mw := range rou.middlewares {
		handler = mw(handler)
	}
	h := otelhttp.NewHandler(handler, "gofr-router")
	rou.Router.Handler(method, pattern, h)
}

func (rou *Router) UseMiddleware(mws ...Middleware) {
	rou.middlewares = append(rou.middlewares, mws...)
}

type staticFileConfig struct {
	directoryName string
}

func (rou *Router) AddStaticFiles(endpoint, dirName string) {
	cfg := staticFileConfig{directoryName: dirName}

	fileServer := http.FileServer(http.Dir(cfg.directoryName))
	rou.Router.HandlerFunc(http.MethodGet, endpoint+"/*filepath", func(w http.ResponseWriter, r *http.Request) {
		// Strip the prefix and serve static files
		http.StripPrefix(endpoint, fileServer).ServeHTTP(w, r)
	})
	//rou.Router.NewRoute().PathPrefix(endpoint + "/").Handler(http.StripPrefix(endpoint, cfg.staticHandler(fileServer)))
}

func (staticConfig staticFileConfig) staticHandler(fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path

		filePath := strings.Split(url, "/")

		fileName := filePath[len(filePath)-1]

		const defaultSwaggerFileName = "openapi.json"

		if _, err := os.Stat(filepath.Clean(filepath.Join(staticConfig.directoryName, url))); fileName == defaultSwaggerFileName && err == nil {
			w.WriteHeader(http.StatusForbidden)

			_, _ = w.Write([]byte("403 forbidden"))

			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
