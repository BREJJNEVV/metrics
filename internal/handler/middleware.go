package handler

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"

	"github.com/BREJJNEVV/metrics/internal/compress"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

func WithLogging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			logger.Info("request completed",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()),
			)
		})
	}

}

func GzipDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := compress.NewReader(r.Body)
			if err != nil {
				http.Error(w, "bad gzip", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		next.ServeHTTP(w, r)
	})
}

type gzipResponceWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			grw := &gzipResponceWriter{
				ResponseWriter: w,
			}
			defer func() {
				if grw.gz != nil {
					grw.gz.Close()
				}
			}()
			next.ServeHTTP(grw, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (grw *gzipResponceWriter) WriteHeader(statusCode int) {
	if grw.wroteHeader {
		return
	}
	grw.wroteHeader = true
	ct := grw.Header().Get("Content-Type")
	if strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html") {
		gz, err := compress.NewWriter(grw.ResponseWriter)
		if err != nil {
			grw.gz = nil
		} else {
			grw.gz = gz
			grw.Header().Set("Content-Encoding", "gzip")
		}
	}
	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponceWriter) Write(data []byte) (int, error) {
	if !grw.wroteHeader {
		grw.WriteHeader(http.StatusOK)
	}
	if grw.gz != nil {
		return grw.gz.Write(data)
	}
	return grw.ResponseWriter.Write(data)
}
