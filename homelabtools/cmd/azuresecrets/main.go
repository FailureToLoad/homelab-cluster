package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/failuretoload/homelabtools/azure"
	"github.com/failuretoload/homelabtools/local"
	"github.com/failuretoload/homelabtools/secret"
)

type ExternalSecretPrincipal struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TenantID     string `json:"tenantId"`
}

func (esp ExternalSecretPrincipal) Validate() error {
	var err error
	if esp.ClientID == "" {
		err = errors.Join(err, errors.New("client id is required"))
	}
	if esp.ClientSecret == "" {
		err = errors.Join(err, errors.New("client secret is required"))
	}
	if esp.TenantID == "" {
		err = errors.Join(err, errors.New("tenant id is required"))
	}

	return err
}

func main() {
	secretPath := flag.String("secretpath", "../cluster/bootstrap/secrets/azure", "directory to generate secrets in")
	flag.Parse()
	if err := run(*secretPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(baseDir string) error {
	ctx := context.Background()

	vaultURL, err := local.GetKeyringValue(secret.VaultURL)
	if err != nil {
		return err
	}

	vc, err := azure.NewVaultClient(ctx, vaultURL)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	externalSecretPrincipal, err := getExternalSecretPrincipal(vc)
	if err != nil {
		return fmt.Errorf("failed to get external secret principal: %w", err)
	}

	if err := writeFile(filepath.Join(baseDir, "client-id"), externalSecretPrincipal.ClientID); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(baseDir, "client-secret"), externalSecretPrincipal.ClientSecret); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(baseDir, "tenant-id"), externalSecretPrincipal.TenantID); err != nil {
		return err
	}

	fmt.Println("Successfully fetched and wrote all secrets")
	return nil
}

func writeFile(path, data string) error {
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

func getExternalSecretPrincipal(vc *azure.VaultClient) (*ExternalSecretPrincipal, error) {
	secretJSON, err := vc.GetSecret("external-secret-principal")
	if err != nil {
		return nil, fmt.Errorf("failed to get external-secret-principal: %w", err)
	}

	if secretJSON != "" {
		var esp ExternalSecretPrincipal
		if err := json.Unmarshal([]byte(secretJSON), &esp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal external-secret-principal: %w", err)
		}
		return &esp, nil
	}

	esp, err := getExternalSecretPrincipalFromKeyring()
	if err != nil {
		return nil, err
	}

	secretData, err := json.Marshal(esp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal external-secret-principal: %w", err)
	}

	if err := vc.SetSecret("external-secret-principal", string(secretData)); err != nil {
		return nil, fmt.Errorf("failed to store external-secret-principal in vault: %w", err)
	}

	return esp, nil
}

func getExternalSecretPrincipalFromKeyring() (*ExternalSecretPrincipal, error) {
	clientID, err := local.GetKeyringValue(secret.ClientID)
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		return nil, fmt.Errorf("client id not found in keyring")
	}

	tenantID, err := local.GetKeyringValue(secret.TenantID)
	if err != nil {
		return nil, err
	}
	if tenantID == "" {
		return nil, fmt.Errorf("tenant id not found in keyring")
	}

	clientSecret, err := local.GetKeyringValue(secret.ClientSecret)
	if err != nil {
		return nil, err
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret not found in keyring")
	}

	return &ExternalSecretPrincipal{
		ClientID:     clientID,
		TenantID:     tenantID,
		ClientSecret: clientSecret,
	}, nil
}
