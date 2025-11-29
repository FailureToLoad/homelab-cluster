package azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type VaultClient struct {
	sc       *azsecrets.Client
	ctx      context.Context
	vaultURL string
}

func NewVaultClient(ctx context.Context, vaultURL string) (*VaultClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	secrets, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets client: %w", err)
	}

	return &VaultClient{
		sc:       secrets,
		ctx:      ctx,
		vaultURL: vaultURL,
	}, nil
}

func (c *VaultClient) SetSecret(name, value string) error {
	_, err := c.sc.SetSecret(c.ctx, name, azsecrets.SetSecretParameters{
		Value: &value,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to set secret %s: %w", name, err)
	}
	return nil
}

func (c *VaultClient) GetSecret(name string) (string, error) {
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
