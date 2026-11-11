package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BREJJNEVV/metrics/internal/sign"
)

func TestHashVerify(t *testing.T) {
	const key = "secret"
	body := []byte(`{"id":"Alloc"}`)
	validHash := sign.Sign(body, key)

	tests := []struct {
		name     string
		hash     string
		body     []byte
		wantCode int
	}{
		{"valid", validHash, body, http.StatusOK},
		{"missing hash", "", body, http.StatusBadRequest},
		{"wrong hash", "deadbeef", body, http.StatusBadRequest},
		{"tampered body", validHash, []byte(`{"id":"Hacked"}`), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			h := HashVerify(key)(next)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tt.body))
			if tt.hash != "" {
				req.Header.Set("HashSHA256", tt.hash)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestHashSign(t *testing.T) {
	const key = "secret"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		w.WriteHeader(http.StatusOK)
	})
	h := HashSign(key)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	gotHash := w.Header().Get("HashSHA256")
	if gotHash == "" {
		t.Fatal("HashSHA256 header missing")
	}
	wantHash := sign.Sign([]byte("hello"), key)
	if gotHash != wantHash {
		t.Errorf("hash mismatch: got %s, want %s", gotHash, wantHash)
	}
	if w.Body.String() != "hello" {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestGzipDecompress(t *testing.T) {
	t.Run("gzipped body", func(t *testing.T) {
		plain := []byte("hello world")
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(plain)
		gz.Close()

		var gotBody []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})
		h := GzipDecompress(next)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if !bytes.Equal(gotBody, plain) {
			t.Errorf("body: got %q, want %q", gotBody, plain)
		}
	})

	t.Run("plain body passes through", func(t *testing.T) {
		plain := []byte("hello")
		var gotBody []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})
		h := GzipDecompress(next)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(plain))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if !bytes.Equal(gotBody, plain) {
			t.Errorf("body: got %q, want %q", gotBody, plain)
		}
	})

	t.Run("bad gzip", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		h := GzipDecompress(next)

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzip"))
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status: got %d, want 400", w.Code)
		}
	})
}

func TestGzipCompress(t *testing.T) {
	t.Run("compresses json", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true}`))
		})
		h := GzipCompress(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("expected gzip encoding, got %q", w.Header().Get("Content-Encoding"))
		}

		gz, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		defer gz.Close()
		body, _ := io.ReadAll(gz)
		if string(body) != `{"ok":true}` {
			t.Errorf("decompressed body: %q", body)
		}
	})

	t.Run("no accept-encoding", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true}`))
		})
		h := GzipCompress(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("should not compress without Accept-Encoding")
		}
		if w.Body.String() != `{"ok":true}` {
			t.Errorf("body: %q", w.Body.String())
		}
	})
}
