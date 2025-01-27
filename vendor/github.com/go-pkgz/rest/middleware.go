package rest

import (
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-pkgz/rest/logger"
)

// Wrap converts a list of middlewares to nested calls (in reverse order)
func Wrap(handler http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	if len(mws) == 0 {
		return handler
	}
	res := handler
	for i := len(mws) - 1; i >= 0; i-- {
		res = mws[i](res)
	}
	return res
}

// AppInfo adds custom app-info to the response header
func AppInfo(app, author, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Author", author)
			w.Header().Set("App-Name", app)
			w.Header().Set("App-Version", version)
			if mhost := os.Getenv("MHOST"); mhost != "" {
				w.Header().Set("Host", mhost)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// Ping middleware response with pong to /ping. Stops chain if ping request detected
func Ping(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/ping") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("pong"))
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Recoverer is a middleware that recovers from panics, logs the panic and returns a HTTP 500 status if possible.
func Recoverer(l logger.Backend) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					l.Logf("request panic, %v", rvr)
					l.Logf(string(debug.Stack()))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Headers middleware adds headers to request
func Headers(headers ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, h := range headers {
				elems := strings.Split(h, ":")
				if len(elems) != 2 {
					continue
				}
				r.Header.Set(strings.TrimSpace(elems[0]), strings.TrimSpace(elems[1]))
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
