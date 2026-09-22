package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func Sign(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))

}

func Verify(data []byte, key string, hash string) bool {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	expected := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(hash))
}
