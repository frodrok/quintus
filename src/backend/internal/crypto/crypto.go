// Package crypto provides AES-256-GCM encryption for DSN storage.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")

// KeyFromHex decodes a 64-character hex string into a 32-byte AES-256 key.
func KeyFromHex(s string) ([32]byte, error) {
	var key [32]byte
	b, err := hex.DecodeString(s)
	if err != nil {
		return key, fmt.Errorf("crypto: key is not valid hex: %w", err)
	}
	if len(b) != 32 {
		return key, fmt.Errorf("crypto: key must be 32 bytes (64 hex chars), got %d", len(b))
	}
	copy(key[:], b)
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM. The returned bytes are
// nonce || ciphertext || tag (standard GCM append).
func Encrypt(key [32]byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: read random nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return sealed, nil
}

// Decrypt decrypts a blob produced by Encrypt.
func Decrypt(key [32]byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, ErrInvalidCiphertext
	}
	nonce, data := ciphertext[:ns], ciphertext[ns:]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	return plain, nil
}