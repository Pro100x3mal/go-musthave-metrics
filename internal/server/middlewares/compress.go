package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// CompressHandler provides HTTP compression middleware.
type CompressHandler struct {
	logger     *zap.Logger
	readerPool *sync.Pool
}

// NewCompressHandler creates a new CompressHandler with the provided logger.
func NewCompressHandler(logger *zap.Logger) *CompressHandler {
	return &CompressHandler{
		logger: logger,
		readerPool: &sync.Pool{
			New: func() interface{} {
				reader, _ := gzip.NewReader(strings.NewReader(""))
				return reader
			},
		},
	}
}

type compressReader struct {
	r    io.ReadCloser
	gzr  *gzip.Reader
	pool *sync.Pool
}

func (ch *CompressHandler) newCompressReader(r io.ReadCloser) (*compressReader, error) {
	gzr := ch.readerPool.Get().(*gzip.Reader)
	if err := gzr.Reset(r); err != nil {
		ch.readerPool.Put(gzr)
		return nil, err
	}

	return &compressReader{
		r:    r,
		gzr:  gzr,
		pool: ch.readerPool,
	}, nil
}

func (cr *compressReader) Read(p []byte) (n int, err error) {
	return cr.gzr.Read(p)
}

func (cr *compressReader) Close() error {
	if err := cr.gzr.Close(); err != nil {
		cr.pool.Put(cr.gzr)
		return err
	}
	cr.pool.Put(cr.gzr)
	return cr.r.Close()
}

type compressWriter struct {
	w                 http.ResponseWriter
	gzw               *gzip.Writer
	shouldCompress    bool
	writeHeaderCalled bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	gzw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)

	return &compressWriter{
		w:   w,
		gzw: gzw,
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

// Middleware provides automatic gzip compression for HTTP responses and decompression for HTTP requests.
// It checks the Accept-Encoding header for gzip support and Content-Encoding header for gzip-compressed requests.
func (ch *CompressHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := ch.newCompressReader(r.Body)
			if err != nil {
				ch.logger.Error("compression error", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer func() {
				if err = cr.Close(); err != nil {
					ch.logger.Error("failed to close gzip reader", zap.Error(err))
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
					ch.logger.Error("failed to close gzip writer", zap.Error(err))
				}
			}()
		}

		next.ServeHTTP(wOut, r)
	})
}
