package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/failuretoload/homelabtools/local"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/role"
)

type Client struct {
	sc       *azsecrets.Client
	ctx      context.Context
	vaultURL string
}

func NewClient(ctx context.Context, vaultURL string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	secrets, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets client: %w", err)
	}

	return &Client{
		sc:       secrets,
		ctx:      ctx,
		vaultURL: vaultURL,
	}, nil
}

func (c *Client) DeleteSecret(name string) error {
	_, err := c.sc.DeleteSecret(c.ctx, name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", name, err)
	}
	return nil
}

func (c *Client) ListSecrets() ([]string, error) {
	var secrets []string
	pager := c.sc.NewListSecretPropertiesPager(nil)

	for pager.More() {
		page, err := pager.NextPage(c.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range page.Value {
			if secret.ID != nil && secret.ID.Name() != "" {
				secrets = append(secrets, secret.ID.Name())
			}
		}
	}

	return secrets, nil
}

func (c *Client) GetClusterSecrets() (*ClusterSecrets, error) {
	return c.GetClusterSecretsWithOverwrite(false)
}

func (c *Client) GetClusterSecretsWithOverwrite(overwrite bool) (*ClusterSecrets, error) {
	var secretJSON string
	var err error

	if !overwrite {
		secretJSON, err = c.getSecret("cluster")
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster secret: %w", err)
		}
	}

	if secretJSON == "" || overwrite {
		bundle, err := generateClusterSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to generate cluster secrets: %w", err)
		}

		// Generate admin client certificate for talosconfig
		adminCert, err := bundle.GenerateTalosAPIClientCertificate(role.MakeSet(role.Admin))
		if err != nil {
			return nil, fmt.Errorf("failed to generate admin certificate: %w", err)
		}

		clusterSecrets := &ClusterSecrets{
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
		}

		secretData, err := json.Marshal(clusterSecrets)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster secrets: %w", err)
		}

		if err := c.setSecret("cluster", string(secretData)); err != nil {
			return nil, fmt.Errorf("failed to store cluster secret: %w", err)
		}

		return clusterSecrets, nil
	}

	var clusterSecrets ClusterSecrets
	if err := json.Unmarshal([]byte(secretJSON), &clusterSecrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster secret: %w", err)
	}

	return &clusterSecrets, nil
}

func (c *Client) GetCiliumSecrets() (*CiliumSecrets, error) {
	secretJSON, err := c.getSecret("cilium")
	if err != nil {
		return nil, fmt.Errorf("failed to get cilium secret: %w", err)
	}

	if secretJSON == "" {
		caCert, caKey, err := generateCACert("Cilium CA")
		if err != nil {
			return nil, fmt.Errorf("failed to generate cilium CA: %w", err)
		}

		ciliumSecrets := &CiliumSecrets{
			CiliumCACRT: caCert,
			CiliumCAKey: caKey,
		}

		secretData, err := json.Marshal(ciliumSecrets)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cilium secrets: %w", err)
		}

		if err := c.setSecret("cilium", string(secretData)); err != nil {
			return nil, fmt.Errorf("failed to store cilium secret: %w", err)
		}

		return ciliumSecrets, nil
	}

	var ciliumSecrets CiliumSecrets
	if err := json.Unmarshal([]byte(secretJSON), &ciliumSecrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cilium secret: %w", err)
	}

	return &ciliumSecrets, nil
}

func (c *Client) GetHubbleSecrets() (*HubbleSecrets, error) {
	secretJSON, err := c.getSecret("hubble")
	if err != nil {
		return nil, fmt.Errorf("failed to get hubble secret: %w", err)
	}

	if secretJSON == "" {
		ciliumSecrets, err := c.GetCiliumSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to get cilium secrets for hubble cert generation: %w", err)
		}

		tlsCert, tlsKey, err := generateTLSCert(ciliumSecrets.CiliumCACRT, ciliumSecrets.CiliumCAKey, "*.hubble-grpc.cilium.io")
		if err != nil {
			return nil, fmt.Errorf("failed to generate hubble TLS cert: %w", err)
		}

		hubbleSecrets := &HubbleSecrets{
			HubbleTLSCRT: tlsCert,
			HubbleTLSKey: tlsKey,
		}

		secretData, err := json.Marshal(hubbleSecrets)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal hubble secrets: %w", err)
		}

		if err := c.setSecret("hubble", string(secretData)); err != nil {
			return nil, fmt.Errorf("failed to store hubble secret: %w", err)
		}

		return hubbleSecrets, nil
	}

	var hubbleSecrets HubbleSecrets
	if err := json.Unmarshal([]byte(secretJSON), &hubbleSecrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hubble secret: %w", err)
	}

	return &hubbleSecrets, nil
}

func (c *Client) GetExternalSecretPrincipal() (*ExternalSecretPrincipal, error) {
	secretJSON, err := c.getSecret("external-secret-principal")
	if err != nil {
		return nil, fmt.Errorf("failed to get external-secret-principal: %w", err)
	}

	if secretJSON != "" {
		var esp ExternalSecretPrincipal
		if err := json.Unmarshal([]byte(secretJSON), &esp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal external-secret-principal: %w", err)
		}
		return &esp, nil
	}

	esp, err := c.getExternalSecretPrincipalFromKeyring()
	if err != nil {
		return nil, err
	}

	secretData, err := json.Marshal(esp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal external-secret-principal: %w", err)
	}

	if err := c.setSecret("external-secret-principal", string(secretData)); err != nil {
		return nil, fmt.Errorf("failed to store external-secret-principal in vault: %w", err)
	}

	return esp, nil
}

func (c *Client) getExternalSecretPrincipalFromKeyring() (*ExternalSecretPrincipal, error) {
	clientID, err := local.GetClientID()
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		return nil, fmt.Errorf("client id not found in keyring")
	}

	tenantID, err := local.GetTenantID()
	if err != nil {
		return nil, err
	}
	if tenantID == "" {
		return nil, fmt.Errorf("tenant id not found in keyring")
	}

	clientSecret, err := local.GetClientSecret()
	if err != nil {
		return nil, err
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret not found in keyring")
	}

	return &ExternalSecretPrincipal{
		ClientID:     clientID,
		TenantID:     tenantID,
		ClientSecret: clientSecret,
	}, nil
}

func (c *Client) setSecret(name, value string) error {
	_, err := c.sc.SetSecret(c.ctx, name, azsecrets.SetSecretParameters{
		Value: &value,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to set secret %s: %w", name, err)
	}
	return nil
}

func (c *Client) getSecret(name string) (string, error) {
	resp, err := c.sc.GetSecret(c.ctx, name, "", nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get secret %s: %w", name, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has no value", name)
	}

	return *resp.Value, nil
}

func generateClusterSecrets() (*secrets.Bundle, error) {
	version, _ := config.ParseContractFromVersion("v1.6.2")
	bundle, err := secrets.NewBundle(secrets.NewClock(), version)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}
