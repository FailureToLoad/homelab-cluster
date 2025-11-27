# Local Cluster Setup

This repo came about by following [this very helpful series of blog posts](https://rcwz.pl/2025-10-04-installing-talos-on-raspberry-pi-5/) by [artuross](https://github.com/artuross). His guide focuses on vendoring manifests for a much higher level of confidence and control when managing a home cluster.  

This guide is much lazier. I just want to run a very slim, lightweight repo that gets me a running cluster.  

Note that I'm not choosing azure keyvault for some kind of secret advantage or preference, I just get dev credits for azure and figured why not make use of them.  

My cluster consists of three 8gb raspberry pi 5's and one 16gb. This is overkill honestly but I have them so why not.  

## Requirements

- A linux repo that supports `secret-tool`
- At least two rpi5's
- An azure subscription (or a friend with one)
- [azureCLI](https://learn.microsoft.com/en-us/cli/azure/what-is-azure-cli?view=azure-cli-latest)
- [talosctl](https://docs.siderolabs.com/talos/v1.9/reference/cli)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)

## Prep/Comments

As of today (2025.11.21) the 10.7 cluster image from the talos-rpi5 repo is stable.

```bash
wget https://github.com/talos-rpi5/talos-builder/releases/download/v1.10.7-rpi5/metal-arm64.raw.zst
unzstd metal-arm64.raw.zst
```

Flash it to all the devices. Note - remember to use lsblk to check for which drive to flash to.  

For flashing an NVME make sure your pi has it set as a boot option first. After that you don't have to do anything fancy, just toss it into an external enclosure, connect it to whichever system is going to be controlling your talos cluster, and flash it as though it was a usb device. I use this [pluggable m2 enclosure](https://a.co/d/h8ynbUA).  

```bash
sudo dd if=metal-arm64.raw of=/dev/sda bs=4M status=progress conv=fsync
```

[You can use kustomize directly](https://www.reddit.com/r/kubernetes/comments/1o1owch/comment/njc8ske/?utm_source=share&utm_medium=web3x&utm_name=web3xcss&utm_term=1&utm_content=share_button) instead of kubesource, but I found kubesource to be pretty convenient.

Also, I'm aware that the usage of the go-keyring isn't correct since I'm setting the user attribute to a random key, but its working fine for me at the moment so I'm quite ready to fix it.  

## Azure Setup

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
