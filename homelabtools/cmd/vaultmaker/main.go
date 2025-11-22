package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/failuretoload/homelabtools/cmd/vaultmaker/internal"
)

func main() {
	config := &internal.Config{
		ResourceGroup:  "",
		KeyVaultName:   "",
		Location:       "",
		SPName:         "",
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		ServiceName:    "",
	}

	if config.SubscriptionID == "" {
		log.Fatal(fmt.Errorf("AZURE_SUBSCRIPTION_ID environment variable must be set"))
	}

	if config.TenantID == "" {
		log.Fatal(fmt.Errorf("AZURE_TENANT_ID environment variable must be set"))
	}

	builder, err := internal.NewEnvironmentBuilder(config)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create manager: %w", err))
	}

	ctx := context.Background()
	if planErr := builder.Plan(ctx); planErr != nil {
		log.Fatal(planErr)
	}

	err = builder.Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
