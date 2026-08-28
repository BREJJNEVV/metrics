package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

const level = gzip.BestCompression

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func NewWriter(w io.Writer) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, level)
}

func NewReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}
