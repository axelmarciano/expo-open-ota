package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("Started %s %s with query: %s and headers: %v", r.Method, r.RequestURI, r.URL.RawQuery, r.Header)

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered during %s %s\nQuery: %s\nHeaders: %v\nError: %v\nStack Trace:\n%s",
					r.Method, r.RequestURI, r.URL.RawQuery, r.Header, err, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(recorder, r)

		if recorder.statusCode >= 500 {
			log.Printf("Error detected: %s %s returned status %d", r.Method, r.RequestURI, recorder.statusCode)
		}
		log.Printf("Completed %s %d in %v", r.RequestURI, recorder.statusCode, time.Since(start))
	})
}
