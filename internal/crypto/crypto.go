package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const Salt = "ai-relay-monitor"

func deriveKey(raw string) ([]byte, error) {
	if len(raw) == 64 {
		key, err := hex.DecodeString(raw)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(key) == 32 {
		return key, nil
	}

	key, err = scrypt.Key([]byte(raw), []byte(Salt), 16384, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
	}
	return key, nil
}

func Encrypt(plaintext, keyRaw string) (string, error) {
	key, err := deriveKey(keyRaw)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return "v1:" + base64.StdEncoding.EncodeToString(result), nil
}

func Decrypt(encrypted, keyRaw string) (string, error) {
	if !strings.HasPrefix(encrypted, "v1:") {
		return "", errors.New("invalid encrypted format: missing v1 prefix")
	}

	key, err := deriveKey(keyRaw)
	if err != nil {
		return "", err
	}

	raw, err := base64.StdEncoding.DecodeString(encrypted[3:])
	if err != nil {
		return "", err
	}

	if len(raw) < 29 {
		return "", errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := raw[:12]
	ciphertext := raw[12:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
