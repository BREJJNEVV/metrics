package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(data []byte, key string) string {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func Verify(data string, key string, hash string) bool {
	h := sha256.New()
	h.Write([]byte(data))
	h.Write([]byte(key))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(hash))
}
