package server

import (
	"net/http"
)

// wrapResponseWriter is a proxy around an http.ResponseWriter that allows you to hook into the response.
type wrapResponseWriter struct {
	http.ResponseWriter

	wroteHeader bool
	code        int
}

func (wrw *wrapResponseWriter) WriteHeader(code int) {
	if !wrw.wroteHeader {
		wrw.code = code
		if code != http.StatusNotFound {
			wrw.wroteHeader = true
			wrw.ResponseWriter.WriteHeader(code)
		}
	}
}

// Write sends bytes to wrapped response writer, in case of not found it suppresses further writes.
func (wrw *wrapResponseWriter) Write(b []byte) (int, error) {
	if wrw.notFound() {
		return len(b), nil
	}
	return wrw.Write(b)
}

func (wrw *wrapResponseWriter) notFound() bool {
	return wrw.code == http.StatusNotFound
}

// fallbackNotFound wraps the given handler with the `fallback` handle to fallback in case of not found.
func fallbackNotFound(handler, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		frw := wrapResponseWriter{ResponseWriter: w}
		handler.ServeHTTP(&frw, r)
		if frw.notFound() {
			w.Header().Del("Content-Type")
			fallback.ServeHTTP(w, r)
		}
	}
}
