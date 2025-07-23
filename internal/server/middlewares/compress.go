package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
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

type compressWriter struct {
	w   http.ResponseWriter
	gzw *gzip.Writer
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
	return cw.gzw.Write(p)
}

func (cw *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		cw.w.Header().Set("Content-Encoding", "gzip")
	}
	cw.w.WriteHeader(statusCode)
}

func (cw *compressWriter) Close() error {
	return cw.gzw.Close()
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

func WithCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer cr.Close()
			r.Body = cr
		}

		wOut := w

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			wOut = cw
			defer cw.Close()
		}

		next.ServeHTTP(wOut, r)
	})
}
