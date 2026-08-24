package githubapp

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ConfigFromEnv reads GitHub App settings from the process environment.
func ConfigFromEnv() (Config, error) {
	c := Config{
		AppID:        os.Getenv("GITHUB_APP_ID"),
		ClientID:     os.Getenv("GITHUB_APP_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_APP_CLIENT_SECRET"),
		Slug:         os.Getenv("GITHUB_APP_SLUG"),
	}
	pemBytes := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY"))
	if pemBytes == "" {
		return c, nil
	}
	key, err := ParsePrivateKey([]byte(pemBytes))
	if err != nil {
		return Config{}, err
	}
	c.PrivateKey = key
	return c, nil
}

// ParsePrivateKey reads a PEM-encoded RSA private key (PKCS1 or PKCS8).
func ParsePrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("github app private key: no PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("github app private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("github app private key: not RSA")
	}
	return key, nil
}
