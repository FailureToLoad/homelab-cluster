package azure

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/failuretoload/homelabtools/convert"
	"github.com/fatih/color"
	"github.com/google/uuid"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
)

type Client struct {
	cred           *azidentity.DefaultAzureCredential
	gc             *graph.GraphServiceClient
	rgc            *armresources.ResourceGroupsClient
	kvc            *armkeyvault.VaultsClient
	rac            *armauthorization.RoleAssignmentsClient
	location       string
	subscriptionID string
	tenantID       string
}

type ServicePrincipalCredentials struct {
	AppID        string
	TenantID     string
	ClientSecret string
}

func New(tenantID, subID, location string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	graphClient, err := graph.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("failed to create graph client: %w", err)
	}

	resourceGroupClient, err := armresources.NewResourceGroupsClient(subID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource groups client: %w", err)
	}

	keyVaultClient, err := armkeyvault.NewVaultsClient(subID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vaults client: %w", err)
	}

	rolesClient, err := armauthorization.NewRoleAssignmentsClient(subID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create role assignments client: %w", err)
	}

	return &Client{
		tenantID:       tenantID,
		subscriptionID: subID,
		location:       location,
		cred:           cred,
		gc:             graphClient,
		rgc:            resourceGroupClient,
		kvc:            keyVaultClient,
		rac:            rolesClient,
	}, nil
}

func (c *Client) CheckResourceGroup(ctx context.Context, resourceGroup string) (bool, error) {
	resp, err := c.rgc.Get(ctx, resourceGroup, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check resource group: %w", err)
	}

	return resp.ID != nil, nil
}

func (c *Client) CreateResourceGroup(ctx context.Context, resourceGroup string) error {
	color.Blue("[INFO] Creating resource group '%s' in location '%s'...", resourceGroup, c.location)

	_, err := c.rgc.CreateOrUpdate(ctx, resourceGroup, armresources.ResourceGroup{
		Location: convert.Pointer(c.location),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create resource group: %w", err)
	}

	color.Green("[✓] Resource group created")
	return nil
}

func (c *Client) CheckKeyVault(ctx context.Context, resourceGroup, vaultName string) (bool, error) {
	resp, err := c.kvc.Get(ctx, resourceGroup, vaultName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check Key Vault: %w", err)
	}

	return resp.ID != nil, nil
}

func (c *Client) CreateKeyVault(ctx context.Context, resourceGroup, vaultName string) (name, kvURL string, err error) {
	name = vaultName
	var poller *runtime.Poller[armkeyvault.VaultsClientCreateOrUpdateResponse]
	for {
		color.Blue("[INFO] Creating Key Vault '%s'...", name)
		color.Blue("[INFO]   - Enabling RBAC authorization")
		color.Blue("[INFO]   - Soft-delete enabled by default (90 day retention)")
		color.Blue("[INFO]   - Enabling purge protection")

		poller, err = c.kvc.BeginCreateOrUpdate(ctx, resourceGroup, name, armkeyvault.VaultCreateOrUpdateParameters{
			Location: convert.Pointer(c.location),
			Properties: &armkeyvault.VaultProperties{
				TenantID: convert.Pointer(c.tenantID),
				SKU: &armkeyvault.SKU{
					Family: convert.Pointer(armkeyvault.SKUFamilyA),
					Name:   convert.Pointer(armkeyvault.SKUNameStandard),
				},
				EnableRbacAuthorization: convert.Pointer(true),
				EnablePurgeProtection:   convert.Pointer(true),
			},
		}, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
				color.Red("[✗] Key Vault name '%s' is already taken (globally)", name)
				newName, promptErr := c.promptForVaultName()
				if promptErr != nil {
					return "", "", fmt.Errorf("failed to get new vault name: %w", promptErr)
				}
				name = newName
				continue
			}
			return name, kvURL, fmt.Errorf("failed to begin create Key Vault: %w", err)
		}

		break
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return name, kvURL, fmt.Errorf("failed to create Key Vault: %w", err)
	}

	color.Green("[✓] Key Vault created")

	kvURL = fmt.Sprintf("https://%s.vault.azure.net/", name)
	if resp.Properties != nil && resp.Properties.VaultURI != nil {
		kvURL = *resp.Properties.VaultURI
	}

	return name, kvURL, nil
}

func (c *Client) CheckServicePrincipal(ctx context.Context, spName string) (string, error) {
	filter := fmt.Sprintf("displayName eq '%s'", spName)
	result, err := c.gc.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: convert.Pointer(filter),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query service principals: %w", err)
	}

	spList := result.GetValue()
	if len(spList) == 0 {
		return "", nil
	}

	appID := spList[0].GetAppId()
	if appID == nil {
		return "", fmt.Errorf("service principal found but has no appId")
	}

	return *appID, nil
}

func (c *Client) CreateServicePrincipal(ctx context.Context, spName string) (*ServicePrincipalCredentials, error) {
	color.Blue("[INFO] Creating service principal '%s'...", spName)

	color.Blue("[INFO] Creating app registration...")
	app := models.NewApplication()
	app.SetDisplayName(convert.Pointer(spName))

	createdApp, err := c.gc.Applications().Post(ctx, app, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	appID := createdApp.GetAppId()
	appObjectID := createdApp.GetId()
	if appID == nil || appObjectID == nil {
		return nil, fmt.Errorf("failed to get app ID or object ID from created application")
	}

	color.Blue("[INFO] Creating service principal for app...")
	sp := models.NewServicePrincipal()
	sp.SetAppId(appID)

	_, err = c.gc.ServicePrincipals().Post(ctx, sp, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create service principal: %w", err)
	}

	color.Blue("[INFO] Creating credentials for service principal...")
	passwordCredential := models.NewPasswordCredential()
	passwordCredential.SetDisplayName(convert.Pointer("Created by homelab setup script"))

	addPasswordBody := applications.NewItemAddPasswordPostRequestBody()
	addPasswordBody.SetPasswordCredential(passwordCredential)

	credResult, err := c.gc.Applications().ByApplicationId(*appObjectID).AddPassword().Post(ctx, addPasswordBody, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	clientSecret := credResult.GetSecretText()
	if clientSecret == nil {
		return nil, fmt.Errorf("failed to get client secret from response")
	}

	creds := &ServicePrincipalCredentials{
		AppID:        *appID,
		TenantID:     c.tenantID,
		ClientSecret: *clientSecret,
	}

	color.Green("[✓] Service principal created (App ID: %s)", *appID)

	return creds, nil
}

func (c *Client) promptForVaultName() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter a new Key Vault name (3-24 alphanumeric characters, globally unique): ")
	vaultName, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	vaultName = strings.TrimSpace(vaultName)
	if len(vaultName) < 3 || len(vaultName) > 24 {
		return "", fmt.Errorf("vault name must be between 3 and 24 characters")
	}

	return vaultName, nil
}

func (c *Client) AssignKeyVaultRoles(ctx context.Context, resourceGroup, kvName, appID string) error {
	vault, err := c.kvc.Get(ctx, resourceGroup, kvName, nil)
	if err != nil {
		return fmt.Errorf("failed to get Key Vault: %w", err)
	}

	if vault.ID == nil {
		return fmt.Errorf("key Vault ID is nil")
	}

	scope := *vault.ID

	filter := fmt.Sprintf("appId eq '%s'", appID)
	spResult, err := c.gc.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: convert.Pointer(filter),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get service principal: %w", err)
	}

	spList := spResult.GetValue()
	if len(spList) == 0 {
		return fmt.Errorf("service principal not found")
	}

	principalID := spList[0].GetId()
	if principalID == nil {
		return fmt.Errorf("service principal ID is nil")
	}

	roles := map[string]string{
		"Key Vault Reader":           "21090545-7ca7-4776-b22c-e363652d74d2",
		"Key Vault Secrets User":     "4633458b-17de-408a-b874-0445c86b69e6",
		"Key Vault Crypto User":      "12338af0-0e69-4776-bea7-57ae8d297424",
		"Key Vault Certificate User": "db79e9a7-68ee-4b58-9aeb-b90e7c24fcba",
	}

	for roleName, roleDefID := range roles {
		color.Blue("[INFO] Assigning role: %s", roleName)

		exists, err := c.checkRoleAssignment(ctx, scope, *principalID, roleDefID)
		if err != nil {
			return fmt.Errorf("failed to check role assignment: %w", err)
		}

		if exists {
			color.Green("[✓] Role '%s' already assigned (skip)", roleName)
			continue
		}

		roleDefinitionID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", c.subscriptionID, roleDefID)
		assignmentName := uuid.NewString()

		_, err = c.rac.Create(ctx, scope, assignmentName, armauthorization.RoleAssignmentCreateParameters{
			Properties: &armauthorization.RoleAssignmentProperties{
				RoleDefinitionID: convert.Pointer(roleDefinitionID),
				PrincipalID:      principalID,
			},
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to assign role '%s': %w", roleName, err)
		}

		color.Green("[✓] Role '%s' assigned", roleName)
	}

	return nil
}

func (c *Client) checkRoleAssignment(ctx context.Context, scope, principalID, roleDefID string) (bool, error) {
	filter := fmt.Sprintf("assignedTo('%s')", principalID)
	pager := c.rac.NewListForScopePager(scope, &armauthorization.RoleAssignmentsClientListForScopeOptions{
		Filter: convert.Pointer(filter),
	})

	roleDefinitionID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", c.subscriptionID, roleDefID)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, assignment := range page.Value {
			if assignment.Properties != nil &&
				assignment.Properties.RoleDefinitionID != nil &&
				*assignment.Properties.RoleDefinitionID == roleDefinitionID {
				return true, nil
			}
		}
	}

	return false, nil
}

func WaitForPropagation(duration time.Duration) {
	color.Blue("[INFO] Waiting for service principal to propagate (%v)...", duration)
	time.Sleep(duration)
}
