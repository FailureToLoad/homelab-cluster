#!/usr/bin/env bash
set -euo pipefail

VAULT_NAME=$(secret-tool lookup service homelab username azure-keyvault-name 2>/dev/null || true)
if [[ -z "$VAULT_NAME" ]]; then
    echo "Error: vault name not found in keyring (service=homelab, username=azure-keyvault-name)"
    exit 1
fi

PRIVATE_KEY_EXISTS=$(az keyvault secret show --vault-name "$VAULT_NAME" --name github-ssh-privatekey --query value -o tsv 2>/dev/null || true)
PUBLIC_KEY_EXISTS=$(az keyvault secret show --vault-name "$VAULT_NAME" --name github-ssh-publickey --query value -o tsv 2>/dev/null || true)

if [[ -n "$PRIVATE_KEY_EXISTS" && -n "$PUBLIC_KEY_EXISTS" ]]; then
    echo "GitHub SSH keys already exist in vault"
    echo "$PUBLIC_KEY_EXISTS"
    exit 0
fi

KEYRING_PRIVATE=$(secret-tool lookup service homelab username github-ssh-privatekey 2>/dev/null || true)
KEYRING_PUBLIC=$(secret-tool lookup service homelab username github-ssh-publickey 2>/dev/null || true)

if [[ -n "$KEYRING_PRIVATE" && -n "$KEYRING_PUBLIC" ]]; then
    echo "Found keys in keyring, uploading to vault..."
    az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-privatekey --value "$KEYRING_PRIVATE" >/dev/null
    az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-publickey --value "$KEYRING_PUBLIC" >/dev/null
    echo "$KEYRING_PUBLIC"
    exit 0
fi

echo "Generating new GitHub SSH keys..."
ssh-keygen -t ed25519 -f /tmp/github_key -N ""

PRIVATE_KEY=$(cat /tmp/github_key)
PUBLIC_KEY=$(cat /tmp/github_key.pub)

rm /tmp/github_key /tmp/github_key.pub

echo "$PRIVATE_KEY" | secret-tool store --label="homelab github-ssh-privatekey" service homelab username github-ssh-privatekey
echo "$PUBLIC_KEY" | secret-tool store --label="homelab github-ssh-publickey" service homelab username github-ssh-publickey

az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-privatekey --value "$PRIVATE_KEY" >/dev/null
az keyvault secret set --vault-name "$VAULT_NAME" --name github-ssh-publickey --value "$PUBLIC_KEY" >/dev/null

echo "$PUBLIC_KEY"