package cluster_test

import (
	"os"
	"testing"

	"github.com/failuretoload/bootstrapper/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testNode struct {
	hostname     string
	address      string
	ephemeralGB  int
	persistentGB int
}

func (n testNode) toNodeConfig() (cluster.NodeConfig, error) {
	return cluster.NewNodeConfig(n.hostname, n.address, cluster.StorageTypeNVMe, n.ephemeralGB, n.persistentGB)
}

func TestNewNodeConfig(t *testing.T) {
	tests := []struct {
		name         string
		hostName     string
		address      string
		ephemeralGB  int
		persistentGB int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid node config",
			hostName:     "control-plane",
			address:      "192.168.1.100",
			ephemeralGB:  150,
			persistentGB: 300,
			expectError:  false,
		},
		{
			name:         "valid with zero volumes",
			hostName:     "worker1",
			address:      "192.168.1.101",
			ephemeralGB:  0,
			persistentGB: 0,
			expectError:  false,
		},
		{
			name:         "invalid - missing hostname",
			hostName:     "",
			address:      "192.168.1.100",
			ephemeralGB:  150,
			persistentGB: 300,
			expectError:  true,
			errorMsg:     "host name is required",
		},
		{
			name:         "invalid - missing address",
			hostName:     "worker1",
			address:      "",
			ephemeralGB:  150,
			persistentGB: 300,
			expectError:  true,
			errorMsg:     "node address is required",
		},
		{
			name:         "invalid - negative ephemeral",
			hostName:     "worker1",
			address:      "192.168.1.101",
			ephemeralGB:  -50,
			persistentGB: 300,
			expectError:  true,
			errorMsg:     "ephemeral volume size cannot be negative",
		},
		{
			name:         "invalid - negative persistent",
			hostName:     "worker1",
			address:      "192.168.1.101",
			ephemeralGB:  150,
			persistentGB: -100,
			expectError:  true,
			errorMsg:     "persistent volume size cannot be negative",
		},
		{
			name:         "invalid - multiple errors",
			hostName:     "",
			address:      "",
			ephemeralGB:  -50,
			persistentGB: -100,
			expectError:  true,
			errorMsg:     "host name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc, err := cluster.NewNodeConfig(tt.hostName, tt.address, cluster.StorageTypeNVMe, tt.ephemeralGB, tt.persistentGB)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, nc)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	validSecrets := cluster.Secrets{
		Token:                     "test-token",
		OSCert:                    "test-os-cert",
		OSKey:                     "test-os-key",
		OSAdminCert:               "test-os-admin-cert",
		OSAdminKey:                "test-os-admin-key",
		ClusterID:                 "test-cluster-id",
		ClusterSecret:             "test-cluster-secret",
		TrustdToken:               "test-trustd-token",
		BootstrapToken:            "test-bootstrap-token",
		SecretBoxEncryptionSecret: "test-secretbox",
		K8SCert:                   "test-k8s-cert",
		K8SKey:                    "test-k8s-key",
		K8SAggregatorCert:         "test-k8s-agg-cert",
		K8SAggregatorKey:          "test-k8s-agg-key",
		K8SServiceAccount:         "test-k8s-sa",
		ECTDCert:                  "test-etcd-cert",
		ECTDKey:                   "test-etcd-key",
		CiliumCACert:              "test-cilium-ca-cert",
		CiliumCAKey:               "test-cilium-ca-key",
		HubbleTLSCert:             "test-hubble-tls-cert",
		HubbleTLSKey:              "test-hubble-tls-key",
	}

	emptySecrets := cluster.Secrets{}

	tests := []struct {
		name          string
		controlPlanes []testNode
		workers       []testNode
		secrets       cluster.Secrets
		expectError   bool
		errorMsg      string
	}{
		{
			name: "valid config with workers",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers: []testNode{
				{"worker1", "192.168.1.101", 100, 200},
				{"worker2", "192.168.1.102", 100, 200},
			},
			secrets:     validSecrets,
			expectError: false,
		},
		{
			name: "valid config with no workers",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers:     []testNode{},
			secrets:     validSecrets,
			expectError: false,
		},
		{
			name: "invalid - empty secrets",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers:     []testNode{{"worker1", "192.168.1.101", 100, 200}},
			secrets:     emptySecrets,
			expectError: true,
			errorMsg:    "token is required",
		},
		{
			name:          "invalid - no control planes",
			controlPlanes: []testNode{},
			workers:       []testNode{{"worker1", "192.168.1.101", 100, 200}},
			secrets:       validSecrets,
			expectError:   true,
			errorMsg:      "at least one control plane is required",
		},
		{
			name: "invalid - duplicate address between nodes",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers:     []testNode{{"worker1", "192.168.1.100", 100, 200}},
			secrets:     validSecrets,
			expectError: true,
			errorMsg:    "duplicate node address",
		},
		{
			name: "invalid - duplicate hostname between nodes",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers:     []testNode{{"cp1", "192.168.1.101", 100, 200}},
			secrets:     validSecrets,
			expectError: true,
			errorMsg:    "duplicate node hostname",
		},
		{
			name: "invalid - duplicate addresses among workers",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers: []testNode{
				{"worker1", "192.168.1.101", 100, 200},
				{"worker2", "192.168.1.101", 100, 200},
			},
			secrets:     validSecrets,
			expectError: true,
			errorMsg:    "duplicate node address",
		},
		{
			name: "invalid - duplicate hostnames among workers",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers: []testNode{
				{"worker1", "192.168.1.101", 100, 200},
				{"worker1", "192.168.1.102", 100, 200},
			},
			secrets:     validSecrets,
			expectError: true,
			errorMsg:    "duplicate node hostname",
		},
		{
			name: "invalid - negative volumes in worker",
			controlPlanes: []testNode{
				{"cp1", "192.168.1.100", 100, 200},
			},
			workers:     []testNode{{"worker1", "192.168.1.101", 100, 200}},
			secrets:     cluster.Secrets{},
			expectError: true,
			errorMsg:    "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlanes := make([]cluster.NodeConfig, 0, len(tt.controlPlanes))
			for _, cp := range tt.controlPlanes {
				nc, cpErr := cp.toNodeConfig()
				require.NoError(t, cpErr)
				controlPlanes = append(controlPlanes, nc)
			}

			workers := make([]cluster.NodeConfig, 0, len(tt.workers))
			for _, w := range tt.workers {
				nc, wErr := w.toNodeConfig()
				require.NoError(t, wErr)
				workers = append(workers, nc)
			}

			cfg, err := cluster.NewConfig("dm-homelab", "192.168.1.100", tt.secrets, controlPlanes, workers)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestGenerateConfigs(t *testing.T) {
	validSecrets := cluster.Secrets{
		Token:                     "test-token",
		OSCert:                    "test-os-cert",
		OSKey:                     "test-os-key",
		OSAdminCert:               "test-os-admin-cert",
		OSAdminKey:                "test-os-admin-key",
		ClusterID:                 "test-cluster-id",
		ClusterSecret:             "test-cluster-secret",
		TrustdToken:               "test-trustd-token",
		BootstrapToken:            "test-bootstrap-token",
		SecretBoxEncryptionSecret: "test-secretbox",
		K8SCert:                   "test-k8s-cert",
		K8SKey:                    "test-k8s-key",
		K8SAggregatorCert:         "test-k8s-agg-cert",
		K8SAggregatorKey:          "test-k8s-agg-key",
		K8SServiceAccount:         "test-k8s-sa",
		ECTDCert:                  "test-etcd-cert",
		ECTDKey:                   "test-etcd-key",
		CiliumCACert:              "test-cilium-ca-cert",
		CiliumCAKey:               "test-cilium-ca-key",
		HubbleTLSCert:             "test-hubble-tls-cert",
		HubbleTLSKey:              "test-hubble-tls-key",
	}

	t.Run("generates control plane config successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		cp, err := cluster.NewNodeConfig("cp1", "192.168.1.100", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		cfg, err := cluster.NewConfig("test-cluster", "192.168.1.100", validSecrets, []cluster.NodeConfig{cp}, nil)
		require.NoError(t, err)

		err = cfg.GenerateConfigs(tmpDir)
		require.NoError(t, err)

		expectedFile := tmpDir + "/test-cluster-cp1-controlplane.yaml"
		_, err = os.Stat(expectedFile)
		require.NoError(t, err, "control plane config file should exist")

		content, err := os.ReadFile(expectedFile)
		require.NoError(t, err)
		assert.NotEmpty(t, content)

		t.Logf("Generated YAML:\n%s", string(content))

		var config map[string]any
		err = yaml.Unmarshal(content, &config)
		require.NoError(t, err)

		machine := config["machine"].(map[string]any)
		network := machine["network"].(map[string]any)
		assert.Equal(t, "cp1", network["hostname"])

		ca := machine["ca"].(map[string]any)
		assert.Equal(t, "dGVzdC1vcy1jZXJ0", ca["crt"])
		assert.Equal(t, "dGVzdC1vcy1rZXk=", ca["key"])
		assert.Equal(t, "test-token", machine["token"])

		cluster := config["cluster"].(map[string]any)
		assert.Equal(t, "test-cluster-id", cluster["id"])
		assert.Equal(t, "test-cluster-secret", cluster["secret"])
		assert.Equal(t, "test-bootstrap-token", cluster["token"])
		assert.Equal(t, "test-secretbox", cluster["secretboxEncryptionSecret"])

		controlPlane := cluster["controlPlane"].(map[string]any)
		assert.Equal(t, "https://192.168.1.100:6443", controlPlane["endpoint"])
		assert.Equal(t, "test-cluster", cluster["clusterName"])

		clusterCA := cluster["ca"].(map[string]any)
		assert.Equal(t, "dGVzdC1rOHMtY2VydA==", clusterCA["crt"])
		assert.Equal(t, "dGVzdC1rOHMta2V5", clusterCA["key"])

		aggregatorCA := cluster["aggregatorCA"].(map[string]any)
		assert.Equal(t, "dGVzdC1rOHMtYWdnLWNlcnQ=", aggregatorCA["crt"])
		assert.Equal(t, "dGVzdC1rOHMtYWdnLWtleQ==", aggregatorCA["key"])

		serviceAccount := cluster["serviceAccount"].(map[string]any)
		assert.Equal(t, "dGVzdC1rOHMtc2E=", serviceAccount["key"])

		apiServer := cluster["apiServer"].(map[string]any)
		certSANs := apiServer["certSANs"].([]any)
		assert.Contains(t, certSANs, "192.168.1.100")

		etcd := cluster["etcd"].(map[string]any)
		etcdCA := etcd["ca"].(map[string]any)
		assert.Equal(t, "dGVzdC1ldGNkLWNlcnQ=", etcdCA["crt"])
		assert.Equal(t, "dGVzdC1ldGNkLWtleQ==", etcdCA["key"])
	})

	t.Run("creates folder if it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedPath := tmpDir + "/nested/folder/path"

		cp, err := cluster.NewNodeConfig("cp1", "192.168.1.100", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		cfg, err := cluster.NewConfig("test-cluster", "192.168.1.100", validSecrets, []cluster.NodeConfig{cp}, nil)
		require.NoError(t, err)

		err = cfg.GenerateConfigs(nestedPath)
		require.NoError(t, err)

		_, err = os.Stat(nestedPath)
		require.NoError(t, err, "nested folder should be created")
	})

	t.Run("generates talosconfig file", func(t *testing.T) {
		tmpDir := t.TempDir()

		cp1, err := cluster.NewNodeConfig("cp1", "192.168.1.100", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		cp2, err := cluster.NewNodeConfig("cp2", "192.168.1.200", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		worker1, err := cluster.NewNodeConfig("worker1", "192.168.1.101", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		worker2, err := cluster.NewNodeConfig("worker2", "192.168.1.102", cluster.StorageTypeNVMe, 100, 200)
		require.NoError(t, err)

		cfg, err := cluster.NewConfig("test-cluster", "192.168.1.100", validSecrets, []cluster.NodeConfig{cp1, cp2}, []cluster.NodeConfig{worker1, worker2})
		require.NoError(t, err)

		err = cfg.GenerateConfigs(tmpDir)
		require.NoError(t, err)

		configPath := tmpDir + "/config"
		_, err = os.Stat(configPath)
		require.NoError(t, err, "config file should exist")

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var talosconfig map[string]any
		err = yaml.Unmarshal(content, &talosconfig)
		require.NoError(t, err)

		assert.Equal(t, "test-cluster", talosconfig["context"])

		contexts := talosconfig["contexts"].(map[string]any)
		clusterContext := contexts["test-cluster"].(map[string]any)

		endpoints := clusterContext["endpoints"].([]any)
		assert.Equal(t, 2, len(endpoints))
		assert.Equal(t, "192.168.1.100", endpoints[0])
		assert.Equal(t, "192.168.1.200", endpoints[1])

		nodes := clusterContext["nodes"].([]any)
		assert.Equal(t, 4, len(nodes))
		assert.Equal(t, "192.168.1.100", nodes[0])
		assert.Equal(t, "192.168.1.200", nodes[1])
		assert.Equal(t, "192.168.1.101", nodes[2])
		assert.Equal(t, "192.168.1.102", nodes[3])

		assert.Equal(t, "dGVzdC1vcy1jZXJ0", clusterContext["ca"])
		assert.Equal(t, "dGVzdC1vcy1hZG1pbi1jZXJ0", clusterContext["crt"])
		assert.Equal(t, "dGVzdC1vcy1hZG1pbi1rZXk=", clusterContext["key"])
	})
}
