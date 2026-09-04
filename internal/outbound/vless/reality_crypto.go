package vless

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/crypto/hkdf"
)

// deriveAuthKey computes the shared secret via X25519 ECDH key exchange between the client's
// private key and the server's public key, then derives a 32-byte authentication key using HKDF-SHA256.
func deriveAuthKey(ecdhe *ecdh.PrivateKey, serverPub []byte, clientRandom []byte) ([]byte, error) {
	if len(clientRandom) < authSaltLen {
		return nil, fmt.Errorf("client random too short: %d (expected at least %d bytes)", len(clientRandom), authSaltLen)
	}

	pub, err := ecdh.X25519().NewPublicKey(serverPub)
	if err != nil {
		return nil, fmt.Errorf("server public key: %w", err)
	}

	sharedSecret, err := ecdhe.ECDH(pub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}

	authKey := make([]byte, authKeyLen)
	r := hkdf.New(sha256.New, sharedSecret, clientRandom[:authSaltLen], []byte(hkdfInfoReality))
	if _, err := r.Read(authKey); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}

	return authKey, nil
}

// sealSessionID encrypts the first 16 bytes of sessionId in-place using AES-256-GCM
// with authKey, nonce, and the raw ClientHello as Additional Authenticated Data (AAD).
func sealSessionID(sessionId []byte, authKey []byte, nonce []byte, helloRaw []byte) ([]byte, error) {
	if len(sessionId) < sessionIDLen {
		return nil, fmt.Errorf("session ID too short: %d (expected at least %d bytes)", len(sessionId), sessionIDLen)
	}

	if len(nonce) < gcmNonceLen {
		return nil, fmt.Errorf("nonce too short: %d (expected at least %d bytes)", len(nonce), gcmNonceLen)
	}

	block, err := aes.NewCipher(authKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	// Seal encrypts sessionId[:16] and appends the 16-byte authentication tag into sessionId[:0]
	aead.Seal(sessionId[:0], nonce, sessionId[:sessionIDPlainLen], helloRaw)
	return sessionId, nil
}

// verifyRealityCert verifies that the leaf certificate presented by the server contains an Ed25519
// public key signed with HMAC-SHA512 using the derived authKey.
func verifyRealityCert(state utls.ConnectionState, authKey []byte) error {
	certs := state.PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("no peer certificates provided")
	}

	leafCert := certs[0]

	pubKey, ok := leafCert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not Ed25519 (fallback mode detected)")
	}

	h := hmac.New(sha512.New, authKey)
	h.Write(pubKey)
	expectedSignature := h.Sum(nil)

	// Constant-time comparison to prevent side-channel timing attacks
	if !hmac.Equal(expectedSignature, leafCert.Signature) {
		return fmt.Errorf("certificate signature mismatch")
	}

	return nil
}
