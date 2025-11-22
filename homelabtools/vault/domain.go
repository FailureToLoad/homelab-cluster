package vault

import "errors"

type ClusterSecrets struct {
	Token                     string `json:"token"`
	OSCert                    string `json:"osCert"`
	OSKey                     string `json:"osKey"`
	OSAdminCert               string `json:"osAdminCert"`
	OSAdminKey                string `json:"osAdminKey"`
	ClusterID                 string `json:"clusterID"`
	ClusterSecret             string `json:"clusterSecret"`
	TrustdToken               string `json:"trustdToken"`
	BootstrapToken            string `json:"bootstrapToken"`
	SecretBoxEncryptionSecret string `json:"secretboxEncryptionSecret"`
	K8SCert                   string `json:"k8sCert"`
	K8SKey                    string `json:"k8sKey"`
	K8SAggregatorCert         string `json:"k8sAggregatorCert"`
	K8SAggregatorKey          string `json:"k8sAggregatorKey"`
	K8SServiceAccount         string `json:"k8sServiceAccount"`
	ECTDCert                  string `json:"etcdCert"`
	ECTDKey                   string `json:"etcdKey"`
}

func (cs ClusterSecrets) Validate() error {
	var err error
	if cs.Token == "" {
		err = errors.Join(err, errors.New("token is required"))
	}
	if cs.OSCert == "" {
		err = errors.Join(err, errors.New("OS certificate is required"))
	}
	if cs.OSKey == "" {
		err = errors.Join(err, errors.New("OS key is required"))
	}
	if cs.OSAdminCert == "" {
		err = errors.Join(err, errors.New("OS admin certificate is required"))
	}
	if cs.OSAdminKey == "" {
		err = errors.Join(err, errors.New("OS admin key is required"))
	}
	if cs.ClusterID == "" {
		err = errors.Join(err, errors.New("cluster ID is required"))
	}
	if cs.ClusterSecret == "" {
		err = errors.Join(err, errors.New("cluster secret is required"))
	}
	if cs.TrustdToken == "" {
		err = errors.Join(err, errors.New("trustd token is required"))
	}
	if cs.BootstrapToken == "" {
		err = errors.Join(err, errors.New("bootstrap token is required"))
	}
	if cs.SecretBoxEncryptionSecret == "" {
		err = errors.Join(err, errors.New("secretbox encryption secret is required"))
	}
	if cs.K8SCert == "" {
		err = errors.Join(err, errors.New("K8s certificate is required"))
	}
	if cs.K8SKey == "" {
		err = errors.Join(err, errors.New("K8s key is required"))
	}
	if cs.K8SAggregatorCert == "" {
		err = errors.Join(err, errors.New("K8s aggregator certificate is required"))
	}
	if cs.K8SAggregatorKey == "" {
		err = errors.Join(err, errors.New("K8s aggregator key is required"))
	}
	if cs.K8SServiceAccount == "" {
		err = errors.Join(err, errors.New("K8s service account is required"))
	}
	if cs.ECTDCert == "" {
		err = errors.Join(err, errors.New("etcd certificate is required"))
	}
	if cs.ECTDKey == "" {
		err = errors.Join(err, errors.New("etcd key is required"))
	}

	return err
}

type CiliumSecrets struct {
	CiliumCACRT string `json:"ciliumCaCrt"`
	CiliumCAKey string `json:"ciliumCaKey"`
}

func (hs CiliumSecrets) Validate() error {
	var err error
	if hs.CiliumCACRT == "" {
		err = errors.Join(err, errors.New("cilium crt is required"))
	}
	if hs.CiliumCAKey == "" {
		err = errors.Join(err, errors.New("cilium key is required"))
	}

	return err
}

type HubbleSecrets struct {
	HubbleTLSCRT string `json:"hubbleTlsCrt"`
	HubbleTLSKey string `json:"hubbleTlsKey"`
}

func (esp HubbleSecrets) Validate() error {
	var err error
	if esp.HubbleTLSCRT == "" {
		err = errors.Join(err, errors.New("hubble tls crt is required"))
	}
	if esp.HubbleTLSKey == "" {
		err = errors.Join(err, errors.New("hubble tls key is required"))
	}

	return err
}

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
