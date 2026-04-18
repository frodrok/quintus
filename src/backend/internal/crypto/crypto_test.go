package crypto_test

import (
	"bytes"
	"testing"

	"github.com/fredrik/quintus/internal/crypto"
)

func TestRoundtrip(t *testing.T) {
	key, err := crypto.KeyFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("postgres://user:password@host:5432/db?sslmode=disable")
	ct, err := crypto.Encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext equals plaintext")
	}
	got, err := crypto.Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q, want %q", got, plain)
	}
}

func TestDecryptTampered(t *testing.T) {
	key, err := crypto.KeyFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	if err != nil {
		t.Fatal(err)
	}
	ct, _ := crypto.Encrypt(key, []byte("secret"))
	ct[len(ct)-1] ^= 0xff // flip last byte
	_, err = crypto.Decrypt(key, ct)
	if err != crypto.ErrInvalidCiphertext {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestKeyFromHexErrors(t *testing.T) {
	_, err := crypto.KeyFromHex("notHex!")
	if err == nil {
		t.Fatal("expected error for non-hex input")
	}
	_, err = crypto.KeyFromHex("deadbeef") // too short
	if err == nil {
		t.Fatal("expected error for short key")
	}
}