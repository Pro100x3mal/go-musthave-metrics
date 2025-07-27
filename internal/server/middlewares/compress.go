package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/logger"
	"go.uber.org/zap"
)

type compressReader struct {
	r   io.ReadCloser
	gzr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:   r,
		gzr: gzr,
	}, nil
}

func (cr *compressReader) Read(p []byte) (n int, err error) {
	return cr.gzr.Read(p)
}

func (cr *compressReader) Close() error {
	if err := cr.r.Close(); err != nil {
		return err
	}
	return cr.gzr.Close()
}

type compressWriter struct {
	w                 http.ResponseWriter
	gzw               *gzip.Writer
	shouldCompress    bool
	writeHeaderCalled bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:   w,
		gzw: gzip.NewWriter(w),
	}
}

func (cw *compressWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw *compressWriter) Write(p []byte) (int, error) {
	if !cw.writeHeaderCalled {
		cw.WriteHeader(http.StatusOK)
	}

	if cw.shouldCompress {
		return cw.gzw.Write(p)
	}

	return cw.w.Write(p)
}

func (cw *compressWriter) WriteHeader(statusCode int) {
	if cw.writeHeaderCalled {
		return
	}

	cw.writeHeaderCalled = true

	isJSON := strings.Contains(cw.w.Header().Get("Content-Type"), "application/json")
	isHTML := strings.Contains(cw.w.Header().Get("Content-Type"), "text/html")

	if statusCode < 300 && (isJSON || isHTML) {
		cw.shouldCompress = true
		cw.w.Header().Set("Content-Encoding", "gzip")
	}

	cw.w.WriteHeader(statusCode)
}

func (cw *compressWriter) Close() error {
	if cw.shouldCompress {
		return cw.gzw.Close()
	}
	return nil
}

func WithCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				logger.Log.Error("compression error", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer func() {
				if err = cr.Close(); err != nil {
					logger.Log.Error("failed to close gzip reader", zap.Error(err))
				}
			}()
			r.Body = cr
		}

		wOut := w

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			wOut = cw
			defer func() {
				if err := cw.Close(); err != nil {
					logger.Log.Error("failed to close gzip writer", zap.Error(err))
				}
			}()
		}

		next.ServeHTTP(wOut, r)
	})
}
