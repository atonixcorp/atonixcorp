package crypto

import "testing"

func TestDecryptAESGCMRejectsCiphertextShorterThanNonce(t *testing.T) {
	if _, err := DecryptAESGCM(make([]byte, 32), []byte("short")); err == nil {
		t.Fatal("DecryptAESGCM accepted ciphertext shorter than its nonce")
	}
}
