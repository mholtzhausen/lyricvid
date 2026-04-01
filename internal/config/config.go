// Package config manages persistent configuration stored in ~/.lyricvid.yaml.
// API keys are stored AES-GCM encrypted; the encryption key is derived from a
// fixed application secret via SHA-256. This prevents casual plaintext reading
// while keeping the config file self-contained (no user password required).
package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// appSecret is the fixed derivation input for the AES key. Changing it
// invalidates all previously stored values.
const appSecret = "lyricvid-config-key-v1"

// Config holds all persisted application settings.
type Config struct {
	GeminiAPIKey string `yaml:"gemini_api_key,omitempty"`
}

// configPath returns the path to ~/.lyricvid.yaml.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".lyricvid.yaml"), nil
}

// Load reads and decrypts the config file. Returns an empty Config if the
// file does not exist.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}

	var raw struct {
		GeminiAPIKey string `yaml:"gemini_api_key,omitempty"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parsing config %q: %w", path, err)
	}

	cfg := Config{}
	if raw.GeminiAPIKey != "" {
		cfg.GeminiAPIKey, err = decrypt(raw.GeminiAPIKey)
		if err != nil {
			return Config{}, fmt.Errorf("decrypting gemini_api_key: %w", err)
		}
	}
	return cfg, nil
}

// Save encrypts sensitive fields and writes the config file.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	raw := struct {
		GeminiAPIKey string `yaml:"gemini_api_key,omitempty"`
	}{}

	if cfg.GeminiAPIKey != "" {
		raw.GeminiAPIKey, err = encrypt(cfg.GeminiAPIKey)
		if err != nil {
			return fmt.Errorf("encrypting gemini_api_key: %w", err)
		}
	}

	data, err := yaml.Marshal(&raw)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config %q: %w", path, err)
	}
	return nil
}

// derivedKey returns the 32-byte AES key derived from appSecret.
func derivedKey() []byte {
	h := sha256.Sum256([]byte(appSecret))
	return h[:]
}

// encrypt encrypts plaintext with AES-GCM and returns a base64-encoded blob
// with the nonce prepended to the ciphertext.
func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(derivedKey())
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
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt reverses encrypt.
func decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(derivedKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (wrong key or corrupted data): %w", err)
	}
	return string(plaintext), nil
}
