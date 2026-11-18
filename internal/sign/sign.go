// Package sign предоставляет функции для вычисления и проверки HMAC-SHA256 подписи данных.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// Sign вычисляет HMAC-SHA256 подпись данных и возвращает её в формате Base64.
func Sign(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))

}

// Verify проверяет соответствие переданной подписи вычисленной HMAC-SHA256 подписи данных.
func Verify(data []byte, key string, hash string) bool {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	expected := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(hash))
}
