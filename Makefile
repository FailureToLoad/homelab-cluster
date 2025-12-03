.PHONY: vaultmaker talosconfigs freshtalosconfigs azuresecrets bootstrap

vaultmaker:
	cd homelabtools && go run ./cmd/vaultmaker/main.go

talosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go
 
freshtalosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go -overwrite

azuresecrets:
	mkdir -p bootstrap/secrets/azure
	SECRETS_PATH=$$(realpath ./bootstrap/secrets/azure) && cd homelabtools && go run ./cmd/azuresecrets $$SECRETS_PATH

bootstrap:
	kubectl apply -f /home/dm/Repos/homelab-cluster/bootstrap/cluster-appset.yaml
