package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"io"
)

type Envelope struct {
	Ciphertext           []byte
	Nonce                []byte
	EncryptedKeyForAlice []byte
	EncryptedKeyForBob   []byte
}

func EncryptForAliceAndBob(plaintext []byte, km KeyManager) (*Envelope, error) {
	// 1. Generate random AES key
	aesKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, err
	}

	// 2. AES-GCM encrypt plaintext
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	alicePub, err := km.GetPublicKey("alice")
	if err != nil {
		return nil, err
	}
	bobPub, err := km.GetPublicKey("bob")
	if err != nil {
		return nil, err
	}

	encKeyAlice, err := EncryptKeyWithRSA(aesKey, alicePub)
	if err != nil {
		return nil, err
	}
	encKeyBob, err := EncryptKeyWithRSA(aesKey, bobPub)
	if err != nil {
		return nil, err
	}

	return &Envelope{
		Ciphertext:           ciphertext,
		Nonce:                nonce,
		EncryptedKeyForAlice: encKeyAlice,
		EncryptedKeyForBob:   encKeyBob,
	}, nil
}

func EncryptKeyWithRSA(plaintext []byte, key *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, key, plaintext, nil)
}
