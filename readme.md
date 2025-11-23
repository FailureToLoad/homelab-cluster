# Local Cluster Setup

This repo came about by following [this very helpful series of blog posts](https://rcwz.pl/2025-10-04-installing-talos-on-raspberry-pi-5/) by [artuross](https://github.com/artuross). His guide is much more general purpose, this repo is my own slimmed-down version that suits my needs.  

Note that I'm not choosing azure keyvault for some kind of secret advantage or preference, I just get dev credits for azure and figured why not make use of them.  

My cluster consists of three 8gb raspberry pi 5's and one 16gb. This is overkill honestly but I have them so why not.  

## Requirements

- A linux machine that supports `secret-tool`
- At least two rpi5's
- An azure subscription (or a friend with one)
- [azureCLI](https://learn.microsoft.com/en-us/cli/azure/what-is-azure-cli?view=azure-cli-latest)
- [talosctl](https://docs.siderolabs.com/talos/v1.9/reference/cli)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)
- [kubesource](https://github.com/artuross/kubesource)

## Prep/Comments

As of today (2025.11.21) the 10.7 cluster image from the talos-rpi5 repo is stable.

```bash
wget https://github.com/talos-rpi5/talos-builder/releases/download/v1.10.2-rpi5-pre3/metal-arm64.raw.zst
unzstd metal-arm64.raw.zst
```

Flash it to all the devices. Note - remember to use lsblk to check for which drive to flash to.  

For flashing an NVME make sure your pi has it set as a boot option first. After that you don't have to do anything fancy, just toss it into an external enclosure, connect it to whichever system is going to be controlling your talos cluster, and flash it as though it was a usb device. I use this [pluggable m2 enclosure](https://a.co/d/h8ynbUA).  

```bash
sudo dd if=metal-arm64.raw of=/dev/sda bs=4M status=progress conv=fsync
```

[You can use kustomize directly](https://www.reddit.com/r/kubernetes/comments/1o1owch/comment/njc8ske/?utm_source=share&utm_medium=web3x&utm_name=web3xcss&utm_term=1&utm_content=share_button) instead of kubesource, but I found kubesource to be pretty convenient.

Also, I'm aware that the usage of the go-keyring isn't correct since I'm setting the username attribute to a secret key, but it's working fine for me at the moment so I'm not ready to fix it.  

## Azure Setup

Refer to commit [b599a7b](https://github.com/FailureToLoad/homelab-cluster/commit/b599a7b02768a2a5264bd2f1483465b797e3554f).  

### Prerequisites

Before running any commands, ensure you have:

1. **Azure CLI installed and authenticated**:

   ```bash
   az login
   ```

2. **Required environment variables**:

   ```bash
   export AZURE_SUBSCRIPTION_ID="your-subscription-id"
   export AZURE_TENANT_ID="your-tenant-id"
   ```

   You can find your subscription ID and tenant ID with:

   ```bash
   az account show --query "{subscriptionId:id, tenantId:tenantId}" -o json
   ```

### Create Azure Infrastructure

Fill out all the environment details in `homelabtools/cmd/vaultmaker/main.go`

Run `vaultmaker` to create the Resource Group, Key Vault, and Service Principal:

```bash
make vaultmaker
```

This will check for and create any missing resources:

- Resource group
- Azure Key Vault
- Service principal
- Key Vault RBAC role assignments for the service principal
- Credentials stored in gnome keyring

If the Key Vault name is already taken globally, you'll be prompted to enter a new name.  
The service principal credentials are stored in your local keyring.  

## Cluster Configuration

Refer to commit [1d1078f](https://github.com/FailureToLoad/homelab-cluster/commit/1d1078fae424a1818d8cc91164c7f0ab06b580eb).  

### Generate Machine Configurations

Make sure to customize the node definitions in `homelabtools/cmd/generate/main.go` before running this or else you'll end up with my local setup.  

```bash
make generate
```

### First Time Setup

Run for each config, matching it to its IP.

```bash
cd ~/.talos
talosctl apply-config \
  --nodes "NODE_IP" \
  --endpoints "NODE_IP" \
  --file "./CONFIG_FILE_NAME.YAML" \
  --insecure
```

### Updating

Run this command on each node to update with new config values.

```bash
cd ~/.talos
talosctl apply-config \
  --nodes "NODE_IP" \
  --endpoints "NODE_IP" \
  --file "./CONFIG_FILE_NAME.YAML"
```

Then bootstrap the cluster and generate the kube config.  

```bash
talosctl bootstrap --nodes "CONTROL_PLANE_IP"
talosctl kubeconfig --nodes "CONTROL_PLANE_IP"
```

## Replace flannel and kube-proxy with cilium

Refer to commit [87370f0](https://github.com/FailureToLoad/homelab-cluster/commit/87370f0fd6e1de36ea6fbfd3566a919d95023425)

After the cluster is running with default CNI (flannel) and kube-proxy, deploy core components:

```bash
bash scripts/bootstrap.sh
```

This will deploy in order:

1. External Secrets CRDs
2. Namespaces
3. Bootstrap secrets (fetched from Azure Key Vault)
4. Cilium CNI with kube-proxy replacement

Wait for Cilium pods to be ready:

```bash
kubectl get pods -n core-cilium -o wide
```

Once all Cilium pods are Running (1/1), remove the default CNI and kube-proxy:

```bash
kubectl -n kube-system delete ds kube-flannel kube-proxy
```

Verify the cluster is healthy:

```bash
kubectl get nodes
```

All nodes should remain Ready with Cilium now handling CNI and service proxying.

## External Secrets Setup

Refer to commit [e398b4b](https://github.com/FailureToLoad/homelab-cluster/commit/e398b4b70994e1dfb5257ec3b6d31271bfa83f61).  

The External Secrets Operator syncs secrets from Azure Key Vault to Kubernetes. After running `bootstrap.sh`, the operator is deployed but requires proper configuration.

### ClusterSecretStore Configuration

The `ClusterSecretStore` connects External Secrets to Azure Key Vault using Service Principal authentication. The critical requirement is that the `tenantId` must be **explicitly specified** in the spec, not just referenced from a secret.

Your ClusterSecretStore should look like:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: azure-keyvault
spec:
  provider:
    azurekv:
      vaultUrl: "https://your-vault-name.vault.azure.net"
      authType: ServicePrincipal
      tenantId: "your-tenant-id-here"  # Must be explicit
      authSecretRef:
        clientId:
          name: azure-secret
          namespace: core-external-secrets
          key: client-id
        clientSecret:
          name: azure-secret
          namespace: core-external-secrets
          key: client-secret
```

### Verify Setup

Check that the ClusterSecretStore is valid:

```bash
kubectl get clustersecretstore azure-keyvault
```

Should show `STATUS: Valid` and `READY: True`.

### External Secrets Troubleshooting

If ExternalSecrets show `SecretSyncedError`:

1. Check ClusterSecretStore status:

   ```bash
   kubectl get clustersecretstore azure-keyvault -o yaml
   ```

2. Verify service principal credentials:

   ```bash
   kubectl get secret azure-secret -n core-external-secrets -o yaml
   ```

3. Check operator logs:

   ```bash
   kubectl logs -n core-external-secrets deployment/external-secrets --tail=50
   ```

Common issues:

- **"invalid tenantID"**: The `tenantId` field is empty or not specified in the ClusterSecretStore spec
- **"Secret does not exist"**: Secrets haven't been created in Azure Key Vault yet (run `make customresourcevalues`)
- **Service principal permissions**: Ensure the service principal has the required RBAC roles (created by `vaultmaker`)

## ArgoCD Setup

Refer to commits..

- [848c485](https://github.com/FailureToLoad/homelab-cluster/commit/848c485e8eb5c6866ec5bc0e3d57309d130e63e4) for initial config.  

ArgoCD provides GitOps continuous deployment for the cluster. It's deployed with authentication disabled and exposed only via Tailscale for secure private access.

### Secret Management

Refer to commit [b68176c](https://github.com/FailureToLoad/homelab-cluster/commit/b68176c8e364ddc2926a107b16654a3159c1d0e4) for configuring argo with external azure secrets.  
ArgoCD requires secrets stored in Azure Key Vault as individual flattened keys (not JSON):

- `argocd-redis-password` - Redis authentication
- `argocd-server-secretkey` - Server secret key
- `argocd-tls-crt` - TLS certificate
- `argocd-tls-key` - TLS private key
- `github-ssh-privatekey` - SSH private key for Git repos
- `github-ssh-publickey` - SSH public key for Git repos

Generate these secrets:

```bash
cd homelabtools
go run ./cmd/customresourcevalues
```

**Important**: Add the generated SSH public key to your GitHub account for repository access.

### Deploy ArgoCD

ArgoCD is deployed as part of the bootstrap process:

```bash
bash bootstrap.sh
```

This applies:

1. ArgoCD CRDs
2. ArgoCD application manifests
3. ExternalSecrets for ArgoCD credentials (redis password, server secret, GitHub SSH key)

### Verify Deployment

Check that all pods are running:

```bash
kubectl get pods -n core-argocd
```

All pods should show `Running` status. If `argocd-server` is in `CrashLoopBackOff`, it likely started before secrets were synced. Restart it:

```bash
kubectl rollout restart deployment/argocd-server -n core-argocd
```

### Access ArgoCD UI

Next up is securing argo's UI with tailscale. Until that's complete the UI can be accessed at http://localhost:8080 via port forward.

```bash
kubectl port-forward -n core-argocd service/argocd-server 8080:80
```

### Configuration

ArgoCD is configured with:

- **Authentication**: Disabled (`server.disable.auth: true`)
- **TLS**: Insecure mode for port-forward (`server.insecure: true`)
- **Dex**: Disabled (SSO not needed)
- **Notifications**: Disabled
- **Redis init**: Disabled (secrets managed by External Secrets)

### ArgoCD Troubleshooting

If ExternalSecrets fail to sync:

1. Verify ClusterSecretStore is ready (see External Secrets Setup section)
2. Check that secrets exist in Azure Key Vault:

   ```bash
   az keyvault secret list --vault-name your-vault-name --query "[?contains(name, 'argocd') || contains(name, 'github')]"
   ```

3. Check ExternalSecret status:

   ```bash
   kubectl get externalsecrets -n core-argocd
   ```

If pods are failing:

1. Check argocd-server logs:

   ```bash
   kubectl logs -n core-argocd deployment/argocd-server
   ```

2. Verify secrets were created:

   ```bash
   kubectl get secrets -n core-argocd
   ```

Common issues:

- **"secret not found"**: ArgoCD server started before ExternalSecrets synced - restart the server deployment
- **ExternalSecrets not syncing**: ClusterSecretStore `tenantId` not properly configured
- **GitHub repository access**: SSH public key not added to GitHub account

## Tailscale Operator Setup

Refer to commit [2d742e1](https://github.com/FailureToLoad/homelab-cluster/commit/2d742e100b216b1433f6bc9af932c563e39c8e86).  
[And the official tailscale docs.](https://tailscale.com/kb/1236/kubernetes-operator)

The Tailscale Kubernetes Operator enables secure access to cluster services over your private Tailscale network (tailnet) without exposing them to the public internet.  

### Tailscale Prerequisites

Before deploying the operator, you need to create OAuth credentials in your Tailscale account:

1. **Configure ACL tags in your tailnet policy file**:

   Navigate to [Access Controls](https://login.tailscale.com/admin/acls) in the Tailscale admin console and add:

   ```json
   {
     "tagOwners": {
       "tag:k8s-operator": [],
       "tag:k8s": ["tag:k8s-operator"]
     }
   }
   ```

2. **Create OAuth client**:

   Go to [Trust Credentials](https://login.tailscale.com/admin/settings/trust-credentials) and create an OAuth client with:
   - **Scopes**: `Devices -> Core` (read,write), `Keys -> Auth Keys` (read,write)
   - **Tags**: `tag:k8s-operator`

3. **Store OAuth credentials in Azure Key Vault**:

   ```bash
   az keyvault secret set --vault-name your-vault-name --name tailscale-oauth-client --value "YOUR_CLIENT_ID"
   az keyvault secret set --vault-name your-vault-name --name tailscale-oauth-secret --value "YOUR_CLIENT_SECRET"
   ```

### Deploy Tailscale Operator

The operator is deployed as part of the bootstrap process:

```bash
make bootstrap
```

This deploys:

1. Tailscale operator CRDs
2. Operator deployment with proper RBAC
3. ExternalSecret to sync OAuth credentials from Azure Key Vault
4. Cilium socket bypass configuration for kube-proxy replacement mode

### Verify Tailscale Deployment

Check that the operator is running:

```bash
kubectl get pods -n core-tailscale
```

The operator pod should show `Running` status.

Verify the operator joined your tailnet:

Go to [Machines](https://login.tailscale.com/admin/machines) in the Tailscale admin console and look for a node named `homelab-k8s-operator` tagged with `tag:k8s-operator`.

### Tailscale Configuration

The operator is configured with:

- **Hostname**: `homelab-k8s-operator`
- **Operator tags**: `tag:k8s-operator`
- **Proxy tags**: `tag:k8s` (for services exposed by the operator)
- **API server proxy**: Disabled (not needed for single-cluster setup)
- **Cilium compatibility**: Socket bypass annotation configured for kube-proxy replacement mode

### Tailscale Operator Troubleshooting

If the operator fails to start:

1. **Check operator logs**:

   ```bash
   kubectl logs -n core-tailscale deployment/operator --tail=50
   ```

2. **Common errors**:
   - **"API token invalid" (401)**: OAuth credentials are incorrect. Verify secrets in Azure Key Vault match your Tailscale OAuth client
   - **"Permission denied"**: OAuth client missing required scopes (`Devices Core`, `Auth Keys`)
   - **"Tag not found"**: ACL tags not configured in tailnet policy file

3. **Verify OAuth secret**:

   ```bash
   kubectl get externalsecret -n core-tailscale
   kubectl get secret operator-oauth -n core-tailscale -o jsonpath='{.data.client_id}' | base64 -d
   ```

4. **Refresh secrets after updating Key Vault**:

   ```bash
   kubectl delete externalsecret tailscale-operator-oauth -n core-tailscale
   kubectl apply -f k8s/core/tailscale/app/resources/ExternalSecret--tailscale-oauth.yaml
   kubectl rollout restart deployment/operator -n core-tailscale
   ```

If the operator isn't appearing in your tailnet:

- Check that ACL tags are configured correctly
- Verify OAuth client has `tag:k8s-operator` assigned
- Check operator logs for authentication errors

### Cilium Compatibility

Since the cluster runs Cilium in kube-proxy replacement mode, the operator deployment includes the annotation:

```yaml
podAnnotations:
  io.cilium.no-track-port: "41641"
```

This enables socket load balancer bypass in the operator's pod namespace, required for Tailscale ingress and egress services to work correctly with Cilium.

### Expose ArgoCD UI over Tailnet

This just requires adding an ingress rule detailed in commit [1e3aa5f](https://github.com/FailureToLoad/homelab-cluster/commit/1e3aa5f6d3c44f6a7a16004fcea5dacc2f68624c). Add the rule and then bootstrap.
