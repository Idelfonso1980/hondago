package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	encryptedPrefix = "enc:v1:"
	secretsEnvKey   = "HONDAGO_SECRETS_KEY"
)

var (
	secretsLoadOnce sync.Once
	secretsKey      []byte
	secretsLoadErr  error
)

func getSecretsKey() ([]byte, error) {
	secretsLoadOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv(secretsEnvKey))
		if raw == "" {
			secretsLoadErr = fmt.Errorf("%s nÃ£o definida", secretsEnvKey)
			return
		}
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			secretsLoadErr = fmt.Errorf("base64 invÃ¡lido em %s: %w", secretsEnvKey, err)
			return
		}
		if len(key) != 32 {
			secretsLoadErr = fmt.Errorf("%s deve ter 32 bytes decodificados (AES-256)", secretsEnvKey)
			return
		}
		secretsKey = key
	})
	if secretsLoadErr != nil {
		return nil, secretsLoadErr
	}
	return secretsKey, nil
}

func encryptAtRest(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	key, err := getSecretsKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAtRest(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.HasPrefix(raw, encryptedPrefix) {
		return raw, nil
	}
	rest := strings.TrimPrefix(raw, encryptedPrefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("ciphertext invÃ¡lido")
	}
	key, err := getSecretsKey()
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
