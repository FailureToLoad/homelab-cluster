# Authentication Services

## LLDAP Azure Secrets

```bash
az keyvault secret set --vault-name kv-name --name lldap-jwt-secret --value "$(openssl rand -base64 32)"
az keyvault secret set --vault-name kv-name --name lldap-key-seed --value "$(openssl rand -base64 32)"
az keyvault secret set --vault-name kv-name --name lldap-admin-user --value 'your-admin-username'
az keyvault secret set --vault-name kv-name --name lldap-admin-password --value 'your-secure-password'
```
