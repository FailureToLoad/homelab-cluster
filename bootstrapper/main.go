package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/failuretoload/bootstrapper/cluster"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/role"
)

const (
	clusterName = "dm-homelab"
	pattern     = "dm-homelab-*.yaml"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

func run() error {
	cp1 := os.Getenv("NODE1")
	if cp1 == "" {
		return fmt.Errorf("NODE1 not set")
	}

	cp2 := os.Getenv("NODE2")
	if cp2 == "" {
		return fmt.Errorf("NODE2 not set")
	}

	cp3 := os.Getenv("NODE3")
	if cp3 == "" {
		return fmt.Errorf("NODE3 not set")
	}

	worker := os.Getenv("NODE4")
	if worker == "" {
		return fmt.Errorf("NODE4 not set")
	}

	talosDir := filepath.Join(os.Getenv("HOME"), ".talos")
	if err := os.MkdirAll(talosDir, 0o700); err != nil {
		return err
	}
	if err := performBackup(talosDir); err != nil {
		return err
	}

	clusterSecrets, err := getClusterSecrets()
	if err != nil {
		return fmt.Errorf("failed to get cluster secrets: %w", err)
	}

	batman, err := cluster.NewNodeConfig(
		"batman",
		cp1,
		cluster.StorageTypeNVMe,
		50,
		150,
	)
	if err != nil {
		return fmt.Errorf("failed to create batman config: %w", err)
	}

	nightwing, err := cluster.NewNodeConfig(
		"nightwing",
		cp2,
		cluster.StorageTypeMMC,
		50,
		300,
	)
	if err != nil {
		return fmt.Errorf("failed to create nightwing config: %w", err)
	}

	redhood, err := cluster.NewNodeConfig(
		"redhood",
		cp3,
		cluster.StorageTypeMMC,
		50,
		150,
	)
	if err != nil {
		return fmt.Errorf("failed to create redhood config: %w", err)
	}

	robin, err := cluster.NewNodeConfig(
		"robin",
		worker,
		cluster.StorageTypeMMC,
		50,
		150,
	)
	if err != nil {
		return fmt.Errorf("failed to create robin config: %w", err)
	}

	controlPlanes := []cluster.NodeConfig{batman, nightwing, redhood}
	workers := []cluster.NodeConfig{robin}

	cfg, err := cluster.NewConfig(clusterName, cp1, *clusterSecrets, controlPlanes, workers)
	if err != nil {
		return fmt.Errorf("failed to create cluster config: %w", err)
	}

	if err := cfg.GenerateConfigs(talosDir); err != nil {
		return fmt.Errorf("failed to generate configs: %w", err)
	}

	if err := saveClusterSecrets(talosDir, clusterSecrets); err != nil {
		return fmt.Errorf("failed to save cluster secrets: %w", err)
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

func getClusterSecrets() (*cluster.Secrets, error) {
	clusterSecretsPath := filepath.Join(os.Getenv("HOME"), ".talos", "cluster.json")
	if data, err := os.ReadFile(clusterSecretsPath); err == nil {
		var cs cluster.Secrets
		if err := json.Unmarshal(data, &cs); err == nil {
			return &cs, nil
		}
	}

	bundle, err := generateClusterSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cluster secrets: %w", err)
	}

	adminCert, err := bundle.GenerateTalosAPIClientCertificate(role.MakeSet(role.Admin))
	if err != nil {
		return nil, fmt.Errorf("failed to generate admin certificate: %w", err)
	}

	ciliumCACert, ciliumCAKey, hubbleTLSCert, hubbleTLSKey, err := cluster.GenerateCiliumSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cilium secrets: %w", err)
	}

	clusterSecrets := &cluster.Secrets{
		Token:                     string(bundle.TrustdInfo.Token),
		OSCert:                    string(bundle.Certs.OS.Crt),
		OSKey:                     string(bundle.Certs.OS.Key),
		OSAdminCert:               string(adminCert.Crt),
		OSAdminKey:                string(adminCert.Key),
		ClusterID:                 bundle.Cluster.ID,
		ClusterSecret:             bundle.Cluster.Secret,
		TrustdToken:               string(bundle.TrustdInfo.Token),
		BootstrapToken:            string(bundle.Secrets.BootstrapToken),
		SecretBoxEncryptionSecret: bundle.Secrets.SecretboxEncryptionSecret,
		K8SCert:                   string(bundle.Certs.K8s.Crt),
		K8SKey:                    string(bundle.Certs.K8s.Key),
		K8SAggregatorCert:         string(bundle.Certs.K8sAggregator.Crt),
		K8SAggregatorKey:          string(bundle.Certs.K8sAggregator.Key),
		K8SServiceAccount:         string(bundle.Certs.K8sServiceAccount.Key),
		ECTDCert:                  string(bundle.Certs.Etcd.Crt),
		ECTDKey:                   string(bundle.Certs.Etcd.Key),
		CiliumCACert:              ciliumCACert,
		CiliumCAKey:               ciliumCAKey,
		HubbleTLSCert:             hubbleTLSCert,
		HubbleTLSKey:              hubbleTLSKey,
	}

	return clusterSecrets, nil
}

func generateClusterSecrets() (*secrets.Bundle, error) {
	version, _ := config.ParseContractFromVersion("v1.6.2")
	bundle, err := secrets.NewBundle(secrets.NewClock(), version)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

func saveClusterSecrets(talosDir string, cs *cluster.Secrets) error {
	data, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}
	path := filepath.Join(talosDir, "cluster.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write secrets: %w", err)
	}
	return nil
}
