package cluster

import (
	"encoding/base64"
	"os"
	"text/template"
)

const controlPlaneTemplate = `version: v1alpha1
debug: false
persist: true
machine:
  type: controlplane
  token: {{.Token}}
  ca:
    crt: "{{.OSCert}}"
    key: "{{.OSKey}}"
  certSANs: []
  kubelet:
    image: ghcr.io/siderolabs/kubelet:v1.30.0
    defaultRuntimeSeccompProfileEnabled: true
    disableManifestsDirectory: true
  network:
    hostname: {{.HostName}}
  install:
    disk: {{.InstallDisk}}
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
  nodeLabels:
    node.kubernetes.io/exclude-from-external-load-balancers: ""
cluster:
  id: {{.ClusterID}}
  secret: {{.ClusterSecret}}
  controlPlane:
    endpoint: https://{{.Address}}:6443
  clusterName: {{.ClusterName}}
  network:
    cni:
      name: none
    dnsDomain: cluster.local
    podSubnets:
      - 10.244.0.0/16
    serviceSubnets:
      - 10.96.0.0/12
  token: {{.BootstrapToken}}
  secretboxEncryptionSecret: {{.SecretBoxEncryptionSecret}}
  ca:
    crt: "{{.K8SCert}}"
    key: "{{.K8SKey}}"
  aggregatorCA:
    crt: "{{.K8SAggregatorCert}}"
    key: "{{.K8SAggregatorKey}}"
  serviceAccount:
    key: "{{.K8SServiceAccount}}"
  apiServer:
    image: registry.k8s.io/kube-apiserver:v1.30.0
    certSANs:
      - {{.Address}}
    disablePodSecurityPolicy: true
    admissionControl:
      - name: PodSecurity
        configuration:
          apiVersion: pod-security.admission.config.k8s.io/v1alpha1
          defaults:
            audit: restricted
            audit-version: latest
            enforce: baseline
            enforce-version: latest
            warn: restricted
            warn-version: latest
          exemptions:
            namespaces:
              - kube-system
            runtimeClasses: []
            usernames: []
          kind: PodSecurityConfiguration
    auditPolicy:
      apiVersion: audit.k8s.io/v1
      kind: Policy
      rules:
        - level: Metadata
  controllerManager:
    image: registry.k8s.io/kube-controller-manager:v1.30.0
  proxy:
    disabled: true
  scheduler:
    image: registry.k8s.io/kube-scheduler:v1.30.0
  discovery:
    enabled: true
    registries:
      kubernetes:
        disabled: true
      service: {}
  etcd:
    ca:
      crt: "{{.ECTDCert}}"
      key: "{{.ECTDKey}}"
  allowSchedulingOnControlPlanes: true
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

func (c Config) generateControlPlaneYAML(outPath string) error {
	tmpl, err := template.New("controlplane").Parse(controlPlaneTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := map[string]any{
		"Token":                     c.secrets.Token,
		"OSCert":                    base64.StdEncoding.EncodeToString([]byte(c.secrets.OSCert)),
		"OSKey":                     base64.StdEncoding.EncodeToString([]byte(c.secrets.OSKey)),
		"HostName":                  c.controlPlane.HostName,
		"InstallDisk":               c.controlPlane.StorageType.InstallDisk(),
		"ClusterID":                 c.secrets.ClusterID,
		"ClusterSecret":             c.secrets.ClusterSecret,
		"ClusterName":               c.clusterName,
		"Address":                   c.controlPlane.Address,
		"BootstrapToken":            c.secrets.BootstrapToken,
		"SecretBoxEncryptionSecret": c.secrets.SecretBoxEncryptionSecret,
		"K8SCert":                   base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SCert)),
		"K8SKey":                    base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SKey)),
		"K8SAggregatorCert":         base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SAggregatorCert)),
		"K8SAggregatorKey":          base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SAggregatorKey)),
		"K8SServiceAccount":         base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SServiceAccount)),
		"ECTDCert":                  base64.StdEncoding.EncodeToString([]byte(c.secrets.ECTDCert)),
		"ECTDKey":                   base64.StdEncoding.EncodeToString([]byte(c.secrets.ECTDKey)),
		"StorageType":               c.controlPlane.StorageType,
		"Ephemeral":                 c.controlPlane.EphemeralGB,
		"Persistent":                c.controlPlane.PersistentGB,
	}

	return tmpl.Execute(f, data)
}
