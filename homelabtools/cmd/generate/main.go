package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/failuretoload/homelabtools/cluster"
	"github.com/failuretoload/homelabtools/local"
	"github.com/failuretoload/homelabtools/vault"
)

const (
	clusterName = "dm-homelab"
	pattern     = "dm-homelab-*.yaml"
)

func main() {
	overwrite := flag.Bool("overwrite", false, "overwrite existing cluster secrets and regenerate")
	flag.Parse()

	if err := run(*overwrite); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

func run(overwrite bool) error {
	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subID == "" {
		return fmt.Errorf("AZURE_SUBSCRIPTION_ID not set")
	}

	cpIP := os.Getenv("CONTROL_PLANE")
	if cpIP == "" {
		return fmt.Errorf("CONTROL_PLANE not set")
	}

	worker1 := os.Getenv("WORKER_1")
	if worker1 == "" {
		return fmt.Errorf("WORKER_1 not set")
	}

	worker2 := os.Getenv("WORKER_2")
	if worker2 == "" {
		return fmt.Errorf("WORKER_2 not set")
	}

	worker3 := os.Getenv("WORKER_3")
	if worker3 == "" {
		return fmt.Errorf("WORKER_3 not set")
	}

	talosDir := filepath.Join(os.Getenv("HOME"), ".talos")
	if err := os.MkdirAll(talosDir, 0o700); err != nil {
		return err
	}

	vaultURL, err := local.GetKeyvaultURL()
	if err != nil {
		return err
	}

	ctx := context.Background()
	vaultClient, err := vault.NewClient(ctx, vaultURL)
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}

	if err := performBackup(talosDir); err != nil {
		return err
	}

	clusterSecrets, err := vaultClient.GetClusterSecretsWithOverwrite(overwrite)
	if err != nil {
		return fmt.Errorf("failed to get cluster secrets: %w", err)
	}

	controlPlane, err := cluster.NewNodeConfig(
		"batcave",
		cpIP,
		cluster.StorageTypeNVMe,
		50,
		150,
	)
	if err != nil {
		return fmt.Errorf("failed to create control plane config: %w", err)
	}

	nightwing, err := cluster.NewNodeConfig(
		"nightwing",
		worker1,
		cluster.StorageTypeMMC,
		50,
		150,
	)
	if err != nil {
		return fmt.Errorf("failed to create nightwing config: %w", err)
	}

	redhood, err := cluster.NewNodeConfig(
		"redhood",
		worker2,
		cluster.StorageTypeMMC,
		50,
		300,
	)
	if err != nil {
		return fmt.Errorf("failed to create redhood config: %w", err)
	}

	robin, err := cluster.NewNodeConfig(
		"robin",
		worker3,
		cluster.StorageTypeMMC,
		30,
		70,
	)
	if err != nil {
		return fmt.Errorf("failed to create robin config: %w", err)
	}

	cfg, err := cluster.NewConfig(
		clusterName,
		controlPlane,
		*clusterSecrets,
		nightwing,
		redhood,
		robin,
	)
	if err != nil {
		return fmt.Errorf("failed to create cluster config: %w", err)
	}

	if err := cfg.GenerateConfigs(talosDir); err != nil {
		return fmt.Errorf("failed to generate configs: %w", err)
	}

	fmt.Printf("generated configs in %s\n", talosDir)
	return nil
}

func performBackup(talosDir string) error {
	stamp := time.Now().Format("2006.01.02")
	backupDir := filepath.Join(talosDir, stamp)
	if needBackup(talosDir) {
		final := uniqueDir(backupDir)
		if err := os.MkdirAll(final, 0o700); err != nil {
			return err
		}
		moveIfExists(filepath.Join(talosDir, "config"), filepath.Join(final, "config"))
		moveGlob(talosDir, pattern, final)
		fmt.Printf("backed up previous configs to %s\n", final)
	}

	return nil
}

func needBackup(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "config")); err == nil {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	return len(matches) > 0
}

func uniqueDir(base string) string {
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s.%d", base, i)
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

func moveIfExists(src, dst string) {
	if fi, err := os.Stat(src); err == nil && !fi.IsDir() {
		os.Rename(src, dst)
	}
}

func moveGlob(root, pattern, dstDir string) {
	files, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return
	}
	for _, f := range files {
		target := filepath.Join(dstDir, filepath.Base(f))
		os.Rename(f, target)
	}
}
