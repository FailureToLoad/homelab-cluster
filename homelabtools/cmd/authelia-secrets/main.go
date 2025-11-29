package main

import (
	"context"
	"fmt"
	"os"

	"github.com/failuretoload/homelabtools/azure"
	"github.com/failuretoload/homelabtools/cmd/authelia-secrets/internal"
	"github.com/failuretoload/homelabtools/local"
	"github.com/failuretoload/homelabtools/secret"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	vaultURL, err := local.GetKeyringValue(secret.VaultURL)
	if err != nil {
		return fmt.Errorf("failed to get vault URL from keyring: %w", err)
	}

	vc, err := azure.NewVaultClient(ctx, vaultURL)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	fmt.Println("Ensuring LLDAP secrets...")

	if err := internal.EnsureLLDAPAdminPassword(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ lldap-admin-password")

	if err := internal.EnsureLLDAPJWTSecret(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ lldap-jwt-secret")

	if err := internal.EnsureLLDAPKeySeed(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ lldap-key-seed")

	fmt.Println("\nEnsuring Authelia secrets...")

	if err := internal.EnsureAutheliaSessionKey(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ authelia-session-key")

	if err := internal.EnsureAutheliaStorageKey(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ authelia-storage-key")

	if err := internal.EnsureAutheliaJWTSecret(vc); err != nil {
		return err
	}
	fmt.Println("  ✓ authelia-jwt-secret")

	fmt.Println("\nAll secrets are configured.")
	return nil
}
