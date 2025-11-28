package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/failuretoload/homelabtools/local"
	"github.com/failuretoload/homelabtools/vault"
)

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

	vaultURL, err := local.GetKeyvaultURL()
	if err != nil {
		return err
	}

	client, err := vault.NewClient(ctx, vaultURL)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	externalSecretPrincipal, err := client.GetExternalSecretPrincipal()
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
