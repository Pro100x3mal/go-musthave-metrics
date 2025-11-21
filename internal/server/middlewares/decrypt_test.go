package middlewares

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Pro100x3mal/go-musthave-metrics/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate private key")
	return privateKey, &privateKey.PublicKey
}

func savePrivateKeyToFile(t *testing.T, privateKey *rsa.PrivateKey, path string) {
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err, "failed to marshal private key")
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	err = os.WriteFile(path, privateKeyPEM, 0600)
	require.NoError(t, err, "failed to write private key to file")
}

func savePublicKeyToFile(t *testing.T, publicKey *rsa.PublicKey, path string) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err, "failed to marshal public key")
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	err = os.WriteFile(path, publicKeyPEM, 0644)
	require.NoError(t, err, "failed to write public key to file")
}

func TestDecryptHandler_Middleware(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		setupKeys      bool
		requestBody    string
		encryptBody    bool
		expectedStatus int
		expectedBody   string
		wantError      bool
	}{
		{
			name:           "successful decryption",
			setupKeys:      true,
			requestBody:    `{"test":"data"}`,
			encryptBody:    true,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"test":"data"}`,
			wantError:      false,
		},
		{
			name:           "no encryption (no private key set)",
			setupKeys:      false,
			requestBody:    `{"test":"data"}`,
			encryptBody:    false,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"test":"data"}`,
			wantError:      false,
		},
		{
			name:           "empty body with encryption enabled",
			setupKeys:      true,
			requestBody:    "",
			encryptBody:    false,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			wantError:      false,
		},
		{
			name:           "invalid encrypted data",
			setupKeys:      true,
			requestBody:    "invalid encrypted data",
			encryptBody:    false,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var privateKey *rsa.PrivateKey
			var publicKey *rsa.PublicKey

			if tt.setupKeys {
				privateKey, publicKey = generateTestKeyPair(t)
			}

			var requestBody []byte
			if tt.encryptBody && publicKey != nil {
				encrypted, err := crypto.Encrypt(publicKey, []byte(tt.requestBody))
				require.NoError(t, err, "failed to encrypt test data")
				requestBody = encrypted
			} else {
				requestBody = []byte(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(requestBody))
			rec := httptest.NewRecorder()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				w.Write(body)
			})

			decryptHandler := NewDecryptHandler(logger, privateKey)
			handler := decryptHandler.Middleware(nextHandler)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if !tt.wantError {
				assert.Equal(t, tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestDecryptHandler_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	privateKey, publicKey := generateTestKeyPair(t)
	savePrivateKeyToFile(t, privateKey, privateKeyPath)
	savePublicKeyToFile(t, publicKey, publicKeyPath)

	loadedPrivateKey, err := crypto.LoadPrivateKey(privateKeyPath)
	require.NoError(t, err)
	loadedPublicKey, err := crypto.LoadPublicKey(publicKeyPath)
	require.NoError(t, err)

	originalData := []byte(`{"metric":"value","data":12345}`)

	encryptedData, err := crypto.Encrypt(loadedPublicKey, originalData)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(encryptedData))
	rec := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, originalData, body, "decrypted data should match original")
		w.WriteHeader(http.StatusOK)
	})

	logger := zap.NewNop()
	decryptHandler := NewDecryptHandler(logger, loadedPrivateKey)
	handler := decryptHandler.Middleware(nextHandler)

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDecryptHandler_WithoutKeys(t *testing.T) {
	logger := zap.NewNop()
	decryptHandler := NewDecryptHandler(logger, nil)

	testData := []byte(`{"test":"unencrypted"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(testData))
	rec := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, testData, body, "data should pass through unchanged")
		w.WriteHeader(http.StatusOK)
	})

	handler := decryptHandler.Middleware(nextHandler)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
