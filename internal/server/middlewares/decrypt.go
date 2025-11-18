package middlewares

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/pkg/crypto"
	"go.uber.org/zap"
)

type DecryptHandler struct {
	logger     *zap.Logger
	privateKey *rsa.PrivateKey
}

func NewDecryptHandler(logger *zap.Logger, privateKey *rsa.PrivateKey) *DecryptHandler {
	return &DecryptHandler{
		logger:     logger,
		privateKey: privateKey,
	}
}

func (dh *DecryptHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dh.privateKey == nil {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			dh.logger.Error("failed to read request body", zap.Error(err))
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		if len(body) > 0 {
			decryptedBody, err := crypto.Decrypt(dh.privateKey, body)
			if err != nil {
				dh.logger.Error("failed to decrypt request body", zap.Error(err))
				http.Error(w, "Failed to decrypt request body", http.StatusBadRequest)
				return
			}
			body = decryptedBody
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))

		next.ServeHTTP(w, r)
	})
}
