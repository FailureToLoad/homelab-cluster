package internal

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/failuretoload/homelabtools/azure"
	"github.com/fatih/color"
	"github.com/zalando/go-keyring"
)

type Config struct {
	ResourceGroup  string
	KeyVaultName   string
	Location       string
	SPName         string
	SubscriptionID string
	TenantID       string
	ServiceName    string
}

type EnvironmentBuilder struct {
	client         *azure.Client
	serviceName    string
	resourceGroup  string
	keyVaultName   string
	location       string
	spName         string
	subscriptionID string
	tenantID       string
	appID          string
	actions        map[int]Action
}

type Action func(context.Context) error

func NewEnvironmentBuilder(config *Config) (*EnvironmentBuilder, error) {
	c, err := azure.New(config.TenantID, config.SubscriptionID, config.Location)
	if err != nil {
		return nil, err
	}

	return &EnvironmentBuilder{
		client:         c,
		serviceName:    config.ServiceName,
		resourceGroup:  config.ResourceGroup,
		keyVaultName:   config.KeyVaultName,
		location:       config.Location,
		spName:         config.SPName,
		subscriptionID: config.SubscriptionID,
		tenantID:       config.TenantID,
	}, nil
}

func (eb *EnvironmentBuilder) actionManager() func(a Action) {
	actionIndex := 0
	if eb.actions == nil {
		eb.actions = make(map[int]Action)
	}

	return func(a Action) {
		actionIndex++
		eb.actions[actionIndex] = a
	}
}

func (eb *EnvironmentBuilder) Plan(ctx context.Context) error {
	printHeader("Azure Key Vault Setup for Homelab")

	color.Blue("[INFO] Checking Azure CLI login status...")
	color.Green("[✓] Logged in to Azure")

	fmt.Println()
	color.Blue("[INFO] Checking for existing resources...")
	fmt.Println()

	addAction := eb.actionManager()
	rgExists, err := eb.client.CheckResourceGroup(ctx, eb.resourceGroup)
	if err != nil {
		return fmt.Errorf("failed to check resource group: %w", err)
	}

	if rgExists {
		color.Green("[✓] Resource group '%s' exists", eb.resourceGroup)
	} else {
		color.Yellow("[!] Resource group '%s' does not exist", eb.resourceGroup)
		addAction(eb.createResourceGroup)

	}

	kvExists, err := eb.client.CheckKeyVault(ctx, eb.resourceGroup, eb.keyVaultName)
	if err != nil {
		return fmt.Errorf("failed to check Key Vault: %w", err)
	}

	if kvExists {
		color.Green("[✓] Key Vault '%s' exists", eb.keyVaultName)
	} else {
		color.Yellow("[!] Key Vault '%s' does not exist", eb.keyVaultName)
		addAction(eb.createKeyVault)
	}

	appID, err := eb.client.CheckServicePrincipal(ctx, eb.spName)
	if err != nil {
		return fmt.Errorf("failre to check service principal: %w", err)
	}

	spExists := appID != ""
	if spExists {
		eb.appID = appID
		color.Green("[✓] Service principal '%s' exists (App ID: %s)", eb.spName, appID)
		addAction(eb.assignRoles)
	} else {
		color.Yellow("[!] Service principal '%s' does not exist", eb.spName)
		addAction(eb.createServicePrincipal)
	}

	fmt.Println()

	printHeader("FLIGHT PLAN")

	color.Blue("Configuration:")
	fmt.Printf("  Resource Group: %s\n", eb.resourceGroup)
	fmt.Printf("  Key Vault Name: %s\n", eb.keyVaultName)
	fmt.Printf("  Location: %s\n", eb.location)
	fmt.Printf("  Service Principal: %s\n", eb.spName)
	fmt.Println()

	color.Blue("Actions to be performed:")

	if rgExists {
		color.Green("  ✓ Resource group '%s' already exists (skip)", eb.resourceGroup)
	} else {
		color.Yellow("  → Create resource group '%s'", eb.resourceGroup)
	}

	if kvExists {
		color.Green("  ✓ Key Vault '%s' already exists (skip)", eb.keyVaultName)
	} else {
		color.Yellow("  → Create Key Vault '%s' with:", eb.keyVaultName)
		fmt.Println("      - RBAC authorization enabled")
		fmt.Println("      - Soft-delete enabled (90 day retention)")
		fmt.Println("      - Purge protection enabled")
	}

	if spExists {
		color.Green("  ✓ Service principal '%s' already exists", eb.spName)
		color.Yellow("  → Will use existing service principal")
	} else {
		color.Yellow("  → Create service principal '%s'", eb.spName)
		color.Yellow("  → Store credentials in GNOME Keyring")
	}

	color.Yellow("  → Assign Key Vault roles to service principal:")
	fmt.Println("      - Key Vault Reader")
	fmt.Println("      - Key Vault Secrets User")
	fmt.Println("      - Key Vault Crypto User")
	fmt.Println("      - Key Vault Certificate User")

	fmt.Println()

	fmt.Print(color.YellowString("Proceed with setup?") + " [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("setup cancelled")
	}

	fmt.Println()
	printHeader("EXECUTING SETUP")

	return nil
}

func (eb *EnvironmentBuilder) Execute(ctx context.Context) error {
	if len(eb.actions) == 0 {
		return fmt.Errorf("no actions queued for execution, did you run Plan()?")
	}

	for _, action := range eb.actions {
		if err := action(ctx); err != nil {
			return err
		}
	}

	eb.displaySummary()
	return nil
}

func (eb *EnvironmentBuilder) createResourceGroup(ctx context.Context) error {
	if err := eb.client.CreateResourceGroup(ctx, eb.resourceGroup); err != nil {
		return err
	}

	if err := keyring.Set(eb.serviceName, "azure-rg-name", eb.resourceGroup); err != nil {
		return fmt.Errorf("failed to store azure-rg-name in keyring: %w", err)
	}

	return nil
}

func (eb *EnvironmentBuilder) createKeyVault(ctx context.Context) error {
	vaultName, kvURL, err := eb.client.CreateKeyVault(ctx, eb.resourceGroup, eb.keyVaultName)

	if err := keyring.Set(eb.serviceName, "azure-keyvault-name", vaultName); err != nil {
		slog.Warn(fmt.Sprintf("unable to set azure-keyvault-name, manually run: secret-tool store --label='Azure Key Vault Name' service homelab username azure-keyvault-name <<< \"%s\"", eb.keyVaultName))
	}

	if err := keyring.Set(eb.serviceName, "azure-keyvault-url", kvURL); err != nil {
		slog.Warn(fmt.Sprintf("unable to set azure-keyvault-url, manually run: secret-tool store --label='Azure Key Vault URL' service homelab username azure-keyvault-url <<< \"%s\"", kvURL))
	}

	return err
}

func (eb *EnvironmentBuilder) assignRoles(ctx context.Context) error {
	color.Blue("[INFO] Assigning roles", eb.appID)

	return eb.client.AssignKeyVaultRoles(ctx, eb.resourceGroup, eb.keyVaultName, eb.appID)
}

func (eb *EnvironmentBuilder) createServicePrincipal(ctx context.Context) error {
	creds, err := eb.client.CreateServicePrincipal(ctx, eb.spName)
	if err != nil {
		return err
	}

	if err := keyring.Set(eb.serviceName, "azure-client-id", creds.AppID); err != nil {
		return fmt.Errorf("failed to store azure-client-id in keyring: %w", err)
	}

	if err := keyring.Set(eb.serviceName, "azure-tenant-id", creds.TenantID); err != nil {
		return fmt.Errorf("failed to store azure-tenant-id in keyring: %w", err)
	}

	if err := keyring.Set(eb.serviceName, "azure-client-secret", creds.ClientSecret); err != nil {
		return fmt.Errorf("failed to store azure-client-secret in keyring: %w", err)
	}

	eb.appID = creds.AppID

	waitForPropagation()

	return eb.client.AssignKeyVaultRoles(ctx, eb.resourceGroup, eb.keyVaultName, eb.appID)
}

func (eb *EnvironmentBuilder) displaySummary() {
	printHeader("SETUP COMPLETE")

	color.Green("✓ Resource Group: %s", eb.resourceGroup)
	color.Green("✓ Key Vault: %s", eb.keyVaultName)
	color.Green("✓ Service Principal: %s (App ID: %s)", eb.spName, eb.appID)

	fmt.Println()
	fmt.Println("Credentials stored in GNOME Keyring:")
	fmt.Println("  - Azure Homelab Agent - Client ID")
	fmt.Println("  - Azure Homelab Agent - Tenant ID")
	fmt.Println("  - Azure Homelab Agent - Client Secret")
	fmt.Println("  - Azure Key Vault Name")
	fmt.Println("  - Azure Key Vault URL")

	fmt.Println()
	color.Blue("Retrieve credentials with:")
	fmt.Println("  secret-tool lookup service homelab key azure-client-id")
	fmt.Println("  secret-tool lookup service homelab key azure-tenant-id")
	fmt.Println("  secret-tool lookup service homelab key azure-client-secret")
	fmt.Println("  secret-tool lookup service homelab key azure-keyvault-name")
	fmt.Println("  secret-tool lookup service homelab key azure-keyvault-url")

	fmt.Println()
	color.Yellow("Note: Role assignments may take a few minutes to propagate.")
	fmt.Println()
}

func printHeader(title string) {
	line := strings.Repeat("=", 63)
	fmt.Println(line)
	fmt.Printf("  %s\n", title)
	fmt.Println(line)
}

func waitForPropagation() {
	duration := 30 * time.Second
	color.Blue("[INFO] Waiting for service principal to propagate (%v)...", duration)
	time.Sleep(duration)
}
