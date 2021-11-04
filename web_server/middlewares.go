package web_server

import (
	"github.com/LeFinal/masc-server/logging"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// LoggingResponseWriter is a minimal wrapper for http.ResponseWriter that
// allows the written HTTP status code to be captured for logging.
type LoggingResponseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader wraps the WriteHeader method from http.ResponseWriter in order to
// record the written status.
func (rw *LoggingResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs the incoming HTTP request, status, method, path and
// duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrappedWriter := &LoggingResponseWriter{
			ResponseWriter: w,
		}
		next.ServeHTTP(wrappedWriter, r)
		logging.WebServerLogger.WithFields(logrus.Fields{
			"status":   wrappedWriter.status,
			"method":   r.Method,
			"path":     r.URL.EscapedPath(),
			"duration": time.Since(start),
		}).Debug(r.URL.String())
	})
}

// noCacheMiddleware forbids caching.
func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Avoid caching.
		w.Header().Set("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
		next.ServeHTTP(w, r)
	})
}
