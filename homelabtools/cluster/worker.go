package cluster

import (
	"encoding/base64"
	"os"
	"text/template"
)

const workerTemplate = `version: v1alpha1
debug: false
persist: true
machine:
  type: worker
  token: {{.Token}}
  ca:
    crt: "{{.OSCert}}"
    key: ""
  certSANs: []
  kubelet:
    image: ghcr.io/siderolabs/kubelet:v1.30.0
    defaultRuntimeSeccompProfileEnabled: true
    disableManifestsDirectory: true
    nodeIP:
      validSubnets:
        - 192.168.50.0/24
  network:
    hostname: {{.HostName}}
  install:
    disk: /dev/{{.StorageType}}
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.10.7
    wipe: false
  features:
    rbac: true
    stableHostname: true
    apidCheckExtKeyUsage: true
    diskQuotaSupport: true
    kubePrism:
      enabled: true
      port: 7445
    hostDNS:
      enabled: true
      forwardKubeDNSToHost: true
cluster:
  id: {{.ClusterID}}
  secret: {{.ClusterSecret}}
  controlPlane:
    endpoint: https://{{.ControlPlaneAddress}}:6443
  clusterName: {{.ClusterName}}
  network:
    dnsDomain: cluster.local
    podSubnets:
      - 10.244.0.0/16
    serviceSubnets:
      - 10.96.0.0/12
  token: {{.BootstrapToken}}
  ca:
    crt: "{{.K8SCert}}"
    key: ""
  discovery:
    enabled: true
    registries:
      kubernetes:
        disabled: true
      service: {}
---
apiVersion: v1alpha1
kind: VolumeConfig
name: EPHEMERAL
provisioning:
  diskSelector:
    match: disk.transport == "{{.StorageType}}"
  minSize: {{.Ephemeral}}GiB
  maxSize: {{.Ephemeral}}GiB
---
apiVersion: v1alpha1
kind: UserVolumeConfig
name: persistent-data
provisioning:
  diskSelector:
    match: disk.transport == "{{.StorageType}}"
  grow: true
  minSize: {{.Persistent}}GiB
`

func (c Config) generateWorkerYAML(outPath string, worker NodeConfig) error {
	tmpl, err := template.New("worker").Parse(workerTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := map[string]any{
		"Token":               c.secrets.Token,
		"OSCert":              base64.StdEncoding.EncodeToString([]byte(c.secrets.OSCert)),
		"HostName":            worker.HostName,
		"StorageType":         worker.StorageType,
		"ClusterID":           c.secrets.ClusterID,
		"ClusterSecret":       c.secrets.ClusterSecret,
		"ControlPlaneAddress": c.controlPlane.Address,
		"ClusterName":         c.clusterName,
		"BootstrapToken":      c.secrets.BootstrapToken,
		"K8SCert":             base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SCert)),
		"Ephemeral":           worker.EphemeralGB,
		"Persistent":          worker.PersistentGB,
	}

	return tmpl.Execute(f, data)
}
