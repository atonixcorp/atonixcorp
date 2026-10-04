package ledger

import (
	"atonixcorp/security/crypto"
)

type Signer interface {
	Sign(identity string, payload []byte) ([]byte, error)
	Verify(identity string, payload, sig []byte) error
}

type CryptoSigner struct {
	KM crypto.KeyManager
}

func NewCryptoSigner(km crypto.KeyManager) *CryptoSigner {
	return &CryptoSigner{KM: km}
}

func (s *CryptoSigner) Sign(identity string, hash []byte) ([]byte, error) {
	return crypto.Sign(identity, hash, s.KM)
}

func (s *CryptoSigner) Verify(identity string, hash, sig []byte) error {
	return crypto.Verify(identity, hash, sig, s.KM)
}
