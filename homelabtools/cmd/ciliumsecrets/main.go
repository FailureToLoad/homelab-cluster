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
	secretPath := flag.String("secretpath", "../cluster/namespaces/secrets/cilium", "directory to generate secrets in")
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

	ciliumSecrets, err := client.GetCiliumSecrets()
	if err != nil {
		return fmt.Errorf("failed to get cilium secrets: %w", err)
	}

	hubbleSecrets, err := client.GetHubbleSecrets()
	if err != nil {
		return fmt.Errorf("failed to get hubble secrets: %w", err)
	}

	if err := writeFile(filepath.Join(baseDir, "ca.crt"), ciliumSecrets.CiliumCACRT); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(baseDir, "ca.key"), ciliumSecrets.CiliumCAKey); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(baseDir, "tls.crt"), hubbleSecrets.HubbleTLSCRT); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(baseDir, "tls.key"), hubbleSecrets.HubbleTLSKey); err != nil {
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
