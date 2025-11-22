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
