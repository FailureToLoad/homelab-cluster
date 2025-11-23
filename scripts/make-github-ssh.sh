#!/usr/bin/env bash
set -euo pipefail
ssh-keygen -t ed25519 -f /tmp/github_key -N ""

VAULT_NAME=$(az keyvault list --query "[0].name" -o tsv)
az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-privatekey --file /tmp/github_key
az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-publickey --file /tmp/github_key.pub

rm /tmp/github_key /tmp/github_key.pub

az keyvault secret show --vault-name "$VAULT_NAME" --name github-ssh-publickey --query value -o tsv