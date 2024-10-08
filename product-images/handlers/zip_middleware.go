package handlers

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type GzipHandler struct{}

// GzipMiddleware compresses HTTP responses using gzip if the client supports it
func (g *GzipHandler) GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// create a gzip-compressed response writer
			wrw := NewWrappedResponseWriter(rw)
			wrw.Header().Set("Content-Encoding", "gzip")

			// pass the wrapped response writer to the next handler
			next.ServeHTTP(wrw, r)
			defer wrw.Flush()

			return
		}

		// if client does not accept gzip, proceed normally
		next.ServeHTTP(rw, r)
	})
}

// WrappedResponseWriter wraps the original ResponseWriter and includes a gzip.Writer
type WrappedResponseWriter struct {
	rw http.ResponseWriter
	gw *gzip.Writer
}

func NewWrappedResponseWriter(rw http.ResponseWriter) *WrappedResponseWriter {
	// creating a new Writer that will write its compressed outputs to 'rw'
	gw := gzip.NewWriter(rw)

	return &WrappedResponseWriter{gw: gw, rw: rw}
}

// Header delegates the Header method to the original ResponseWriter
func (wrw *WrappedResponseWriter) Header() http.Header {
	return wrw.rw.Header()
}

// Write compresses data before writing it to the original ResponseWriter
func (wrw *WrappedResponseWriter) Write(d []byte) (int, error) {
	return wrw.gw.Write(d)
}

// WriteHeader delegates the WriteHeader method to the original ResponseWriter
func (wrw *WrappedResponseWriter) WriteHeader(statusCode int) {
	wrw.rw.WriteHeader(statusCode)
}

// Flush ensures that all compressed data is sent and the gzip.Writer is closed
func (wrw *WrappedResponseWriter) Flush() {
	wrw.gw.Flush()
	wrw.gw.Close()
}
