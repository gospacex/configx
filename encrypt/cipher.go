// Package encrypt defines a small Cipher interface used by configx to
// transparently decrypt values that look like enc(<ciphertext>).
//
// Implementations:
//
//	AESGCM   – AES-256-GCM with a 32-byte key; ciphertexts are base64 encoded
//	           and prefix with a 12-byte random nonce.
package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Cipher abstracts symmetric encryption/decryption of string values.
type Cipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// EncMarker is the regex used to detect encrypted values inside a config
// document. Any string that matches enc(<payload>) will be passed through
// the configured Cipher's Decrypt method exactly once during loading.
var EncMarker = regexp.MustCompile(`^enc\((.+)\)$`)

// IsEncrypted reports whether s carries the enc(...) wrapper.
func IsEncrypted(s string) bool { return EncMarker.MatchString(s) }

// Unwrap strips the enc(...) wrapper and returns the inner payload.
// It panics-free: returns ("", false) when s is not wrapped.
func Unwrap(s string) (string, bool) {
	m := EncMarker.FindStringSubmatch(s)
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

// Wrap returns enc(payload).
func Wrap(payload string) string { return "enc(" + payload + ")" }

// --- AES-GCM implementation ---------------------------------------------------

type aesGCM struct {
	aead cipher.AEAD
}

// NewAESGCM constructs a Cipher backed by AES-GCM. The key must be 16, 24,
// or 32 bytes long, selecting AES-128, AES-192, or AES-256 respectively.
func NewAESGCM(key []byte) (Cipher, error) {
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("encrypt: aes key must be 16/24/32 bytes, got %d", l)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &aesGCM{aead: aead}, nil
}

func (a *aesGCM) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, a.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := a.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (a *aesGCM) Decrypt(ciphertext string) (string, error) {
	ciphertext = strings.TrimSpace(ciphertext)
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("encrypt: base64 decode: %w", err)
	}
	ns := a.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("encrypt: ciphertext too short")
	}
	nonce, body := raw[:ns], raw[ns:]
	pt, err := a.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("encrypt: gcm open: %w", err)
	}
	return string(pt), nil
}
