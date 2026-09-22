package sign

import (
	"testing"
)

func TestSign(t *testing.T) {
	data := "testData"
	hash1 := Sign([]byte(data), "key")
	hash2 := Sign([]byte(data), "key")

	if hash1 != hash2 {
		t.Errorf("Sign is not deterministic: %q != %q", hash1, hash2)
	}
	if len(hash1) != 44 {
		t.Errorf("expected 44 base64 chars, got %d", len(hash1))
	}
	data2 := "anotherTestData"
	hash3 := Sign([]byte(data2), "key")
	if hash1 == hash3 {
		t.Error("different data produced the same hash")
	}
}

func TestVerify(t *testing.T) {
	data := []byte("testData")
	key := "key"
	hash := Sign([]byte(data), key)

	if !Verify(data, key, hash) {
		t.Error("Verify should return true for valid data")
	}
	if Verify([]byte("otherData"), key, hash) {
		t.Error("Verify should return false for tampered data")
	}
	if Verify(data, "wrongkey", hash) {
		t.Error("Verify should return false for wrong key")
	}
	if Verify(data, key, "wronghash") {
		t.Error("Verify should return false for wrong hash")
	}
}
