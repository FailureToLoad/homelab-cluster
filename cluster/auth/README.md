# Authentication Services

## LLDAP Azure Secrets

```bash
az keyvault secret set --vault-name kv-name --name lldap-jwt-secret --value "$(openssl rand -base64 32)"
az keyvault secret set --vault-name kv-name --name lldap-key-seed --value "$(openssl rand -base64 32)"
az keyvault secret set --vault-name kv-name --name lldap-admin-user --value 'your-admin-username'
az keyvault secret set --vault-name kv-name --name lldap-admin-password --value 'your-secure-password'
```

## Authelia Azure Secrets

```bash
az keyvault secret set --vault-name kv-name --name authelia-jwt-secret --value "$(openssl rand -base64 64)"
az keyvault secret set --vault-name kv-name --name authelia-session-secret --value "$(openssl rand -base64 64)"
az keyvault secret set --vault-name kv-name --name authelia-storage-encryption-key --value "$(openssl rand -base64 64)"
az keyvault secret set --vault-name kv-name --name authelia-oidc-hmac-secret --value "$(openssl rand -base64 64)"
openssl genrsa -out /tmp/authelia-rsa-private.pem 2048
az keyvault secret set --vault-name kv-name --name authelia-oidc-rsa-private-key --file /tmp/authelia-rsa-private.pem
rm /tmp/authelia-rsa-private.pem
openssl ecparam -name prime256v1 -genkey -noout -out /tmp/authelia-ec-private.pem
az keyvault secret set --vault-name kv-name --name authelia-oidc-ec-private-key --file /tmp/authelia-ec-private.pem
rm /tmp/authelia-ec-private.pem
az keyvault secret set --vault-name kv-name --name authelia-datamonster-client-secret --value "$(openssl rand -base64 64)"
az keyvault secret set --vault-name kv-name --name authelia-postgres-password --value "$(openssl rand -base64 32)"
```

### Setup Steps

1. Log into LLDAP at `https://lldap` with your admin credentials
2. Create a new service user in LLDAP:
   - Click "Create User"
   - Username: `authelia`
   - Email: `authelia@homelab.local`
   - Set a secure password (different from your admin password)
3. Store that service account password in Azure Key Vault:

   ```bash
   az keyvault secret set --vault-name kv-name --name authelia-ldap-password --value 'password-you-set-for-authelia-user'
   ```

4. Create the other secrets in Azure Key Vault using the commands above
5. Commit and push changes to trigger ArgoCD sync
