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
  certSANs:
{{.CertSANs}}  kubelet:
    image: ghcr.io/siderolabs/kubelet:v1.34.0
    defaultRuntimeSeccompProfileEnabled: true
    disableManifestsDirectory: true
  network:
    hostname: {{.HostName}}
  time:
    servers:
      - time.cloudflare.com
  install:
    disk: {{.InstallDisk}}
    image: ghcr.io/failuretoload/talos-rpi5-v1.11.5-1-custom:latest
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
    endpoint: https://{{.ControlPlaneEndpoint}}:6443
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
    image: registry.k8s.io/kube-apiserver:v1.34.0
    certSANs:
{{.APICertSANs}}
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
    image: registry.k8s.io/kube-controller-manager:v1.34.0
  proxy:
    disabled: true
  scheduler:
    image: registry.k8s.io/kube-scheduler:v1.34.0
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
  inlineManifests:
    - name: cilium
      contents: |-
` + ciliumManifest + `
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

func (c Config) generateControlPlaneYAML(outPath string, controlPlane NodeConfig) error {
	tmpl, err := template.New("controlplane").Parse(controlPlaneTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	certSANs := c.buildCertSANs()
	apiCertSANs := c.buildAPICertSANs()

	data := map[string]any{
		"Token":                     c.secrets.Token,
		"OSCert":                    base64.StdEncoding.EncodeToString([]byte(c.secrets.OSCert)),
		"OSKey":                     base64.StdEncoding.EncodeToString([]byte(c.secrets.OSKey)),
		"CertSANs":                  certSANs,
		"HostName":                  controlPlane.HostName,
		"InstallDisk":               controlPlane.StorageType.InstallDisk(),
		"ClusterID":                 c.secrets.ClusterID,
		"ClusterSecret":             c.secrets.ClusterSecret,
		"ClusterName":               c.clusterName,
		"ControlPlaneEndpoint":      c.controlPlaneEndpoint,
		"BootstrapToken":            c.secrets.BootstrapToken,
		"SecretBoxEncryptionSecret": c.secrets.SecretBoxEncryptionSecret,
		"K8SCert":                   base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SCert)),
		"K8SKey":                    base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SKey)),
		"K8SAggregatorCert":         base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SAggregatorCert)),
		"K8SAggregatorKey":          base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SAggregatorKey)),
		"K8SServiceAccount":         base64.StdEncoding.EncodeToString([]byte(c.secrets.K8SServiceAccount)),
		"APICertSANs":               apiCertSANs,
		"ECTDCert":                  base64.StdEncoding.EncodeToString([]byte(c.secrets.ECTDCert)),
		"ECTDKey":                   base64.StdEncoding.EncodeToString([]byte(c.secrets.ECTDKey)),
		"StorageType":               controlPlane.StorageType,
		"Ephemeral":                 controlPlane.EphemeralGB,
		"Persistent":                controlPlane.PersistentGB,
		"CiliumCACert":              base64.StdEncoding.EncodeToString([]byte(c.secrets.CiliumCACert)),
		"CiliumCAKey":               base64.StdEncoding.EncodeToString([]byte(c.secrets.CiliumCAKey)),
		"HubbleTLSCert":             base64.StdEncoding.EncodeToString([]byte(c.secrets.HubbleTLSCert)),
		"HubbleTLSKey":              base64.StdEncoding.EncodeToString([]byte(c.secrets.HubbleTLSKey)),
	}

	return tmpl.Execute(f, data)
}

func (c Config) buildCertSANs() string {
	if len(c.controlPlanes) == 0 {
		return ""
	}
	result := ""
	for _, cp := range c.controlPlanes {
		result += "    - " + cp.Address + "\n"
	}
	return result
}

func (c Config) buildAPICertSANs() string {
	if len(c.controlPlanes) == 0 {
		return ""
	}
	result := ""
	result += "      - " + c.controlPlaneEndpoint + "\n"
	for _, cp := range c.controlPlanes {
		result += "      - " + cp.Address + "\n"
	}
	return result
}
