package middlewares

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type LoggerHandler struct {
	logger *zap.Logger
}

func NewLoggerHandler(logger *zap.Logger) *LoggerHandler {
	return &LoggerHandler{
		logger: logger,
	}
}

type responseData struct {
	status int
	size   int
}

func newResponseData() *responseData {
	return &responseData{
		status: http.StatusOK,
		size:   0,
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
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

func (lh *LoggerHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lh.logger.Info("incoming HTTP request",
			zap.String("url", r.URL.String()),
			zap.String("method", r.Method),
			zap.String("remote_addr", r.RemoteAddr),
		)

		respData := newResponseData()
		loggingWriter := newLoggingResponseWriter(w, respData)

		next.ServeHTTP(loggingWriter, r)

		duration := time.Since(start)

		lh.logger.Info("outgoing HTTP response",
			zap.Int("status", respData.status),
			zap.Int("size", respData.size),
			zap.Duration("duration", duration),
		)
	})
}
