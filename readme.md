# Local Cluster Setup

This repo came about by following [this very helpful series of blog posts](https://rcwz.pl/2025-10-04-installing-talos-on-raspberry-pi-5/) by [artuross](https://github.com/artuross). His guide focuses on vendoring manifests for a much higher level of confidence and control when managing a home cluster.  

This guide is much lazier. I just want to run a very slim, lightweight repo that gets me a running cluster.  

Note that I'm not choosing azure keyvault for some kind of secret advantage or preference, I just get dev credits for azure and figured why not make use of them.  

My cluster consists of three 8gb raspberry pi 5's and one 16gb. This is overkill honestly but I have them so why not.  

## Requirements

- A linux repo that supports `secret-tool`
- At least two rpi5's
- [talosctl](https://docs.siderolabs.com/talos/v1.9/reference/cli)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)

## Prep/Comments

I got pretty tired of having to patch my nodes after deploying them, so I set up an actions workflow to just created the ready-to-use base image. Check my [releases page](https://github.com/FailureToLoad/homelab-cluster/releases/) for base images that already have the [linux-tools](github.com/siderolabs/extensions/pkgs/container/util-linux-tools) and [iscsi-tools](https://github.com/siderolabs/extensions/pkgs/container/util-linux-tools) extensions installed.

```bash
wget https://github.com/FailureToLoad/homelab-cluster/releases/download/v1.11.5-1/metal-arm64.raw.zst
unzstd metal-arm64.raw.zst
```

Flash it to all the devices. Note - remember to use lsblk to check for which drive to flash to.  

For flashing an NVME make sure your pi has it set as a boot option first. After that you don't have to do anything fancy, just toss it into an external enclosure, connect it to whichever system is going to be controlling your talos cluster, and flash it as though it was a usb device. I use this [pluggable m2 enclosure](https://a.co/d/h8ynbUA).  

```bash
sudo dd if=metal-arm64.raw of=/dev/sda bs=4M status=progress conv=fsync
```

Also, I'm aware that the usage of the go-keyring isn't correct since I'm setting the user attribute to a random key, but its working fine for me at the moment so I'm not quite ready to fix it.  

## Cluster Configuration

References

- [Part 1 of artuross's blog series](https://rcwz.pl/2025-10-04-installing-talos-on-raspberry-pi-5/)

### Generate Machine Configurations

Customize the node definitions in `homelabtools/cmd/configmaker/main.go`, then run `make bootstrap`. I embedded the cilium manifests into the config template so you won't need to patch that after the fact. This does make first time start-up take longer though.

The values with be stored at `~/.talos/cluster.json` and re-used until deleted. This allows for regenerating configs to modify values without overwriting certs.  

### First Time Setup

Run for each config, matching it to its IP.

```bash
cd ~/.talos
talosctl apply-config --nodes "NODE_IP" --endpoints "NODE_IP" --file "./CONFIG_FILE_NAME.YAML" --insecure
```

After all control-plane configs are applied, bootstrap the cluster once against a single control-plane node (do not repeat on other CPs). Then generate a kubeconfig that targets all control-plane endpoints (or your VIP) so kubectl keeps working if one node is down. I'm using made up IPs as an example here:  

```bash
talosctl bootstrap --nodes "192.168.5.1" --endpoints "192.168.5.1"
talosctl kubeconfig --nodes "192.168.5.1" --endpoints "192.168.5.1,192.168.5.3"
```

Once bootstrapped, set Talos to talk to all control planes (or your control-plane VIP) for day-to-day commands:

```bash
talosctl config endpoints "192.168.5.1" "192.168.5.3"
```

For convenience, I keep a `Makefile` in my .talos directory. That's mostly because I have a tendency to tinker and subsequently ruin my cluster when the urge strikes me.

```Makefile
.PHONY: batman nightwing redhood robin setup batman-update nightwing-update redhood-update robin-update update bootstrap

batman:
 @echo "Applying config to batman (control plane) at $$NODE1..."
 talosctl apply-config --nodes "$$NODE1" --endpoints "$$NODE1" --file "./dm-homelab-batman-controlplane.yaml" --insecure

nightwing:
 @echo "Applying config to nightwing (control plane) at $$NODE2..."
 talosctl apply-config --nodes "$$NODE2" --endpoints "$$NODE2" --file "./dm-homelab-nightwing-controlplane.yaml" --insecure
 
redhood:
 @echo "Applying config to redhood (control plane) at $$NODE3..."
 talosctl apply-config --nodes "$$NODE3" --endpoints "$$NODE3" --file "./dm-homelab-redhood-controlplane.yaml" --insecure

robin:
 @echo "Applying config to robin (worker) at $$NODE4..."
 talosctl apply-config --nodes "$$NODE4" --endpoints "$$NODE4" --file "./dm-homelab-robin-worker.yaml" --insecure

bootstrap:
 @echo "30 second grace period"
 sleep 30
 @echo "Bootstrapping etcd on batman..."
 talosctl bootstrap --nodes "$$NODE1" --endpoints "$$NODE1"
 @echo "10 second grace period"
 sleep 10
 @echo "Configuring kubeconfig and endpoints..."
 talosctl kubeconfig --nodes "$$NODE1" --endpoints "$$NODE1,$$NODE2,$$NODE3"
 talosctl config endpoints "$$NODE1" "$$NODE2" "$$NODE3"
 @echo "Cluster bootstrapped"
 @echo "Check cluster health with: talosctl health"

setup: batman nightwing redhood robin bootstrap

batman-update:
 @echo "Applying config to batman (control plane) at $$NODE1..."
 talosctl apply-config --nodes "$$NODE1" --endpoints "$$NODE1" --file "./dm-homelab-batman-controlplane.yaml" 

nightwing-update:
 @echo "Applying config to nightwing (control plane) at $$NODE2..."
 talosctl apply-config --nodes "$$NODE2" --endpoints "$$NODE2" --file "./dm-homelab-nightwing-controlplane.yaml" 
 
redhood-update:
 @echo "Applying config to redhood (control plane) at $$NODE3..."
 talosctl apply-config --nodes "$$NODE3" --endpoints "$$NODE3" --file "./dm-homelab-redhood-controlplane.yaml" 

robin-update:
 @echo "Applying config to robin (worker) at $$NODE4..."
 talosctl apply-config --nodes "$$NODE4" --endpoints "$$NODE4" --file "./dm-homelab-robin-worker.yaml" 

update: batman-update nightwing-update redhood-update robin-update
 @echo "Update commands issued"
```

## Cert Manager

I don't enjoy managing raw certs so I usually try to get cert-manager in ASAP.  

Run `make certmanager` to install it to your cluster and have it assume responsibility of cilium/hubble certs. It does the following:

1. Deploys cert manager
1. Annotates the previously inline cilium secrets
1. Annotates the previously inline hubble secrets
1. Deploys a cluster issuer resource for cilium's certs which uses the annotations to assume ownership going forward

## External Secrets

I previously was using azure keyvault but it caused me to spend a bit too much time in entra-id than I wanted for a hobby. I moved over to Bitwarden Secret Manager instead.

### Vault Setup

Before running any commands, ensure you have the bitwarden secrets manager CLI installed.

[Relevant docs are here](https://bitwarden.com/help/secrets-manager-cli/)

Since I'm on linux I'll give the short and sweet details.

- Go to the [releases page](https://github.com/bitwarden/sdk-sm/releases) of their github repo and download the latest `unknown-linux-gnu` zip that matches your architecture. For me that was `bws-x86_64-unknown-linux-gnu-1.0.0.zip`.
- It just contains the executable, extract or move it to ~/.local/bin and ensure that location is in your path variable

For now you should make the following on the secret manager website.

1. A project.
1. A machine account.
1. A test secret. I called mine testsecret with a value of 'secret value'.

Make the project first, then jot down its UUID once its made.  
Make the machine account next and jot down its access token once it it made. Make sure to
associate the machine account with the project.  
After the machine account exists, click on it and go to the config tab. Jot down the org id.

Set them in your `.bashrc`

```bash
export BWS_ACCESS_TOKEN=""
export BWS_PROJECT_ID=""
export BWS_ORG_ID=""
```

Source your bashrc or open a new terminal.

### External-Secrets Deployment

Its fairly turn-key, run `make externalsecrets` from the root of the repo.  

This will perform the following:

1. Create the external-secrets namespace
1. Create a tls cert for the bitwarden sdk server
1. Deploy the chart and crds
1. Manually create the token the secret store needs to connect to bitwarden
1. Stand up a ClusterSecretStore for bitwarden and connect it to a ca provider

The TLS cert and CA provider were what I tripped over the first time around on this.  

### Verify External Secrets

Run the following in terminal to connect an external secret to your test secret

```bash
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: testsecret
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: bitwarden-secrets
    kind: ClusterSecretStore
  target:
    name: testsecret
  data:
  - secretKey: value
    remoteRef:
      key: testsecret
EOF
```

Run `kubectl get externalsecret testsecret -n default` to make sure the secret was synced correctly. It might take a few seconds.  

Run `kubectl get secret testsecret -n default -o jsonpath='{.data.value}' | base64 -d` to see the value of the secret.  

Run `kubectl delete externalsecret testsecret -n default` to delete the test secret.

## Tailscale Operator Setup

The Tailscale Kubernetes Operator enables secure access to cluster services over your tailnet.

### Tailscale Prerequisites

Before deploying the operator, you need to create OAuth credentials in your Tailscale account:

1. Configure ACL tags in your tailnet policy file

   Navigate to [Access Controls](https://login.tailscale.com/admin/acls) in the Tailscale admin console and add:

   ```json
   {
     "tagOwners": {
       "tag:k8s-operator": [],
       "tag:k8s": ["tag:k8s-operator"]
     }
   }
   ```

1. Create OAuth client:

   Go to [Trust Credentials](https://login.tailscale.com/admin/settings/trust-credentials) and create an OAuth client with:
   - **Scopes**: `Devices -> Core` (read,write), `Keys -> Auth Keys` (read,write)
   - **Tags**: `tag:k8s-operator`

1. Store OAuth credentials in Bitwarden Secrets Manager:

Store the client id as `tailscale-oauth-client`  
Store the client secret as `tailscale-oauth-secret`  

### Deploy Tailscale Operator

Run `make tailscale`.  

The operator is deployed as part of the bootstrap process:

This deploys:

1. Tailscale operator CRDs
1. ExternalSecret to sync OAuth credentials from Azure Key Vault
1. Cilium socket bypass configuration for kube-proxy replacement mode

### Verify Tailscale Deployment

Check that the operator is running:

```bash
kubectl get pods -n tailscale
```

The operator pod should show `Running` status.

Verify the operator joined your tailnet:

Go to [Machines](https://login.tailscale.com/admin/machines) in the Tailscale admin console and look for a node named `homelab-k8s-operator` tagged with `tag:k8s-operator`.

### Tailscale Operator Troubleshooting

If the operator fails to start:

1. Check operator logs:

   ```bash
   kubectl logs -n tailscale deployment/operator --tail=50
   ```

2. Common errors:
   - **"API token invalid" (401)**: OAuth credentials are incorrect. Verify secrets in Azure Key Vault match your Tailscale OAuth client
   - **"Permission denied"**: OAuth client missing required scopes (`Devices Core`, `Auth Keys`)
   - **"Tag not found"**: ACL tags not configured in tailnet policy file

3. Verify OAuth secret:

   ```bash
   kubectl get externalsecret -n tailscale
   kubectl get secret operator-oauth -n tailscale -o jsonpath='{.data.client_id}' | base64 -d
   ```

4. Refresh secrets after updating Key Vault:

   ```bash
   kubectl delete externalsecret tailscale-operator-oauth -n tailscale
   kubectl apply -f core/apps/tailscale/operator-oauth.yaml
   kubectl rollout restart deployment/operator -n tailscale
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
