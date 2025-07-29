package middlewares

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func newResponseData() *responseData {
	return &responseData{
		status: http.StatusOK,
		size:   0,
	}
}

func newLoggingResponseWriter(w http.ResponseWriter, respData *responseData) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		responseData:   respData,
	}
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			respData := newResponseData()
			loggingWriter := newLoggingResponseWriter(w, respData)

			next.ServeHTTP(loggingWriter, r)

			duration := time.Since(start)

			logger.Info("incoming HTTP request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
			)

			logger.Info("outgoing HTTP response",
				zap.Int("status", respData.status),
				zap.Int("size", respData.size),
			)
		})
	}
}
