package internal

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/failuretoload/homelabtools/local"
	"github.com/failuretoload/homelabtools/secret"
)

type Vault interface {
	GetSecret(name string) (string, error)
	SetSecret(name, value string) error
}

func EnsureAutheliaSessionKey(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.AutheliaSessionKey)
}

func EnsureAutheliaStorageKey(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.AutheliaStorageKey)
}

func EnsureAutheliaJWTSecret(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.AutheliaJWTSecret)
}

func EnsureLLDAPAdminPassword(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.LldapAdminPassword)
}

func EnsureLLDAPJWTSecret(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.LldapJWTSecret)
}

func EnsureLLDAPKeySeed(vc Vault) error {
	return ensureAutheliaSecret(vc, secret.LldapKeySeed)
}

func ensureAutheliaSecret(vc Vault, key string) error {
	vaultValue, err := vc.GetSecret(key)
	if err != nil {
		return err
	}

	if vaultValue != "" {
		return local.SetKeyringValue(key, vaultValue)
	}

	keyringValue, err := local.GetKeyringValue(key)
	if err == nil && keyringValue != "" {
		return vc.SetSecret(key, keyringValue)
	}

	newValue, err := generateRandomString(64)
	if err != nil {
		return err
	}

	if err := local.SetKeyringValue(key, newValue); err != nil {
		return err
	}

	return vc.SetSecret(key, newValue)
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
