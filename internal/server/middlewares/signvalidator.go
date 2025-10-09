package middlewares

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type SignHandler struct {
	logger *zap.Logger
	key    string
}

func NewSignHandler(logger *zap.Logger, key string) *SignHandler {
	return &SignHandler{
		logger: logger,
		key:    key,
	}
}

type signResponseWriter struct {
	http.ResponseWriter
	body              []byte
	status            int
	writeHeaderCalled bool
}

func newSignResponseWriter(w http.ResponseWriter) *signResponseWriter {
	return &signResponseWriter{
		ResponseWriter: w,
		body:           make([]byte, 0),
		status:         http.StatusOK,
	}
}

func (srw *signResponseWriter) Write(p []byte) (int, error) {
	if !srw.writeHeaderCalled {
		srw.WriteHeader(http.StatusOK)
	}
	srw.body = append(srw.body, p...)
	return len(p), nil
}

func (srw *signResponseWriter) WriteHeader(statusCode int) {
	if srw.writeHeaderCalled {
		return
	}
	srw.status = statusCode
	srw.writeHeaderCalled = true
}

func signBody(body []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return h.Sum(nil)
}

func (sh *SignHandler) Middleware(next http.Handler) http.Handler {
	if sh.key == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if receivedHMACStr := r.Header.Get("HashSHA256"); receivedHMACStr != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				sh.logger.Error("failed to read request body", zap.Error(err))
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			expectedHMAC := signBody(body, sh.key)
			receivedHMAC, err := hex.DecodeString(receivedHMACStr)
			if err != nil {
				sh.logger.Error("failed to decode signature", zap.Error(err))
				http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
				return
			}
			if !hmac.Equal(expectedHMAC, receivedHMAC) {
				http.Error(w, "Invalid signature", http.StatusBadRequest)
				return
			}
		}
		srw := newSignResponseWriter(w)

		next.ServeHTTP(srw, r)

		responseHMAC := signBody(srw.body, sh.key)
		w.Header().Set("HashSHA256", hex.EncodeToString(responseHMAC))
		w.WriteHeader(srw.status)
		if len(srw.body) > 0 {
			w.Write(srw.body)
		}
	})
}
