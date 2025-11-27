.PHONY: vaultmaker talosconfigs freshtalosconfigs deploy ciliumsecrets cilium 

vaultmaker:
	cd homelabtools && go run ./cmd/vaultmaker/main.go

talosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go
 
freshtalosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go -overwrite

ciliumsecrets:
	mkdir -p cluster/namespaces/secrets/cilium
	SECRETS_PATH=$$(realpath ./cluster/namespaces/secrets/cilium) && cd homelabtools && go run ./cmd/ciliumsecrets $$SECRETS_PATH

cilium:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/cilium --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status daemonset/cilium -n core-cilium --timeout=120s