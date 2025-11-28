.PHONY: vaultmaker talosconfigs freshtalosconfigs ciliumsecrets cilium azuresecrets external-secrets

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

azuresecrets:
	mkdir -p cluster/namespaces/secrets/azure
	SECRETS_PATH=$$(realpath ./cluster/namespaces/secrets/azure) && cd homelabtools && go run ./cmd/azuresecrets $$SECRETS_PATH

external-secrets:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/external-secrets --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status deployment/external-secrets -n core-external-secrets --timeout=120s
	kubectl apply -k cluster/apps/external-secrets/stores --server-side