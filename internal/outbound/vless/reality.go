package vless

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/hkdf"
)

func deriveAuthKey(ecdhe *ecdh.PrivateKey, serverPub []byte, clientRandom []byte) ([]byte, error) {
	if len(clientRandom) < 20 {
		return nil, fmt.Errorf("clientRandom must be at least 20 bytes")
	}

	pub, err := ecdh.X25519().NewPublicKey(serverPub)
	if err != nil {
		return nil, fmt.Errorf("server public key: %w", err)
	}

	sharedSecret, err := ecdhe.ECDH(pub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}

	authKey := make([]byte, 32)
	r := hkdf.New(sha256.New, sharedSecret, clientRandom[:20], []byte("REALITY"))
	if _, err := r.Read(authKey); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}

	return authKey, nil
}

func sealSessionID(sessionId []byte, authKey []byte, nonce []byte, helloRaw []byte) ([]byte, error) {
	block, err := aes.NewCipher(authKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	aead.Seal(sessionId[:0], nonce, sessionId[:16], helloRaw)
	return sessionId, nil
}
