package local

import (
	"github.com/zalando/go-keyring"
)

const (
	serviceName     = "homelab"
	tenantIDKey     = "azure-tenant-id"
	kvNameKey       = "azure-keyvault-name"
	kvURLKey        = "azure-keyvault-name"
	clientIDKey     = "azure-client-id"
	clientSecretKey = "azure-client-secret"
	rgNameKey       = "azure-rg-name"
)

func SetKeyvaultName(vaultName string) error {
	return setKeyringValue(kvNameKey, vaultName)
}

func GetKeyvaultName() (string, error) {
	return getKeyringValue(kvNameKey)
}

func SetKeyvaultURL(kvURL string) error {
	return setKeyringValue(kvURLKey, kvURL)
}

func GetKeyvaultURL() (string, error) {
	return getKeyringValue(kvURLKey)
}

func SetClientID(clientID string) error {
	return setKeyringValue(clientIDKey, clientID)
}

func GetClientID() (string, error) {
	return getKeyringValue(clientIDKey)
}

func SetClientSecret(clientSecret string) error {
	return setKeyringValue(clientSecretKey, clientSecret)
}

func GetClientSecret() (string, error) {
	return getKeyringValue(clientSecretKey)
}

func SetTenantID(tenantID string) error {
	return setKeyringValue(tenantIDKey, tenantID)
}

func GetTenantID() (string, error) {
	return getKeyringValue(tenantIDKey)
}

func SetResourceGroupName(rgName string) error {
	return setKeyringValue(rgNameKey, rgName)
}

func GetResourceGroupName() (string, error) {
	return getKeyringValue(rgNameKey)
}

func setKeyringValue(key, value string) error {
	return keyring.Set(serviceName, key, value)
}

func getKeyringValue(key string) (string, error) {
	return keyring.Get(serviceName, key)
}
