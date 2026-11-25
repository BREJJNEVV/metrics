package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/BREJJNEVV/metrics/internal/compress"
	"github.com/BREJJNEVV/metrics/internal/sign"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

// WithLogging оборачивает хендлер, добавляя логгирование запросов.
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

// GzipDecompress оборачивает хендлер, добавляя распаковку тела ответа по алгоритму gzip.
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

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

// GzipCompress оборачивает хендлер, добавляя сжатие тела запроса по алгоритму gzip.
func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			grw := &gzipResponseWriter{
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

// HashVerify оборачивает хендлер, добавляя верификацию по алгоритму HashSHA256.
func HashVerify(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newHash := r.Header.Get("HashSHA256")
			if newHash == "" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cannot read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if !sign.Verify(body, key, newHash) {
				http.Error(w, "hash mismatch", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HashSign оборачивает хендлер, добавляя вычисление и установку подписи HashSHA256 в заголовок ответа.
func HashSign(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srw := &signResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(srw, r)
			body := srw.buf.Bytes()
			w.Header().Set("HashSHA256", sign.Sign(body, key))
			w.WriteHeader(srw.status)
			_, _ = w.Write(body)
		})
	}
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
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

func (grw *gzipResponseWriter) Write(data []byte) (int, error) {
	if !grw.wroteHeader {
		grw.WriteHeader(http.StatusOK)
	}
	if grw.gz != nil {
		return grw.gz.Write(data)
	}
	return grw.ResponseWriter.Write(data)
}

type signResponseWriter struct {
	http.ResponseWriter
	buf         bytes.Buffer
	status      int
	wroteHeader bool
}

func (srw *signResponseWriter) WriteHeader(statusCode int) {
	if srw.wroteHeader {
		return
	}
	srw.status = statusCode
	srw.wroteHeader = true
}
func (srw *signResponseWriter) Write(data []byte) (int, error) {
	if !srw.wroteHeader {
		srw.WriteHeader(http.StatusOK)
	}
	return srw.buf.Write(data)
}
