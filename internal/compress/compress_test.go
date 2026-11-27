package compress

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func BenchmarkCompress(b *testing.B) {
	data := []byte("random data for benchmarking comress func")
	b.ResetTimer()
	for b.Loop() {
		_, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	d := []byte("random data for benchmarking comress func")
	data, _ := Compress(d)
	b.ResetTimer()
	for b.Loop() {
		_, err := Decompress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDecompressInvalid(t *testing.T) {
	data := []byte("this is not gzip")
	_, err := Decompress(data)
	if err == nil {
		t.Fatal("data is not gzip, error not recognized, got nil")
	}
}

func TestCompress(t *testing.T) {

	cases := []string{
		"",
		"a",
		"test1312312data123123",
		strings.Repeat("abc", 1000),
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			data := []byte(c)
			compressed, err := Compress(data)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}
			restored, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}
			if !bytes.Equal(data, restored) {
				t.Errorf("round trip mismatch: got %q, want %q", restored, data)
			}
		})
	}

}

func TestNewWriterReader(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	data := []byte("hello gzip")
	if _, writeErr := w.Write(data); writeErr != nil {
		t.Fatalf("Write failed: %v", writeErr)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	r, newReaderErr := NewReader(&buf)
	if newReaderErr != nil {
		t.Fatalf("NewReader failed: %v", newReaderErr)
	}
	restored, readAllErr := io.ReadAll(r)
	if readAllErr != nil {
		t.Fatalf("ReadAll failed: %v", readAllErr)
	}
	if !bytes.Equal(data, restored) {
		t.Errorf("mismatch: got %q, want %q", restored, data)
	}
}
