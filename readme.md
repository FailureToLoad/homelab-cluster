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
