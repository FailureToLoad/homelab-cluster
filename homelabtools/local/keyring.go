package local

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "homelab"

func SetKeyringValue(key, value string) error {
	return keyring.Set(serviceName, key, value)
}

func GetKeyringValue(key string) (string, error) {
	return keyring.Get(serviceName, key)
}
