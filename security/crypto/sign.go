package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

func Sign(identity string, payload []byte, km KeyManager) ([]byte, error) {
	key, err := km.GetPrivateKey(identity)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
}

func Verify(identity string, payload, sig []byte, km KeyManager) error {
	key, err := km.GetPublicKey(identity)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payload)
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sig)
}
