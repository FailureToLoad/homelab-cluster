.PHONY: vaultmaker talosconfigs freshtalosconfigs ciliumsecrets cilium cert-manager azuresecrets external-secrets tailscale argocd

vaultmaker:
	cd homelabtools && go run ./cmd/vaultmaker/main.go

talosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go
 
freshtalosconfigs:
	cd homelabtools && go run ./cmd/configmaker/main.go -overwrite

cilium:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/cilium --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status daemonset/cilium -n core-cilium --timeout=120s

cert-manager:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/cert-manager --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status deployment/cert-manager -n core-cert-manager --timeout=120s
	kubectl rollout status deployment/cert-manager-webhook -n core-cert-manager --timeout=120s
	kubectl rollout status deployment/cert-manager-cainjector -n core-cert-manager --timeout=120s
	sleep 10
	kubectl apply -k cluster/apps/cert-manager/issuers --server-side

azuresecrets:
	mkdir -p cluster/namespaces/secrets/azure
	SECRETS_PATH=$$(realpath ./cluster/namespaces/secrets/azure) && cd homelabtools && go run ./cmd/azuresecrets $$SECRETS_PATH

external-secrets:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/external-secrets --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status deployment/external-secrets -n core-external-secrets --timeout=120s
	kubectl apply -k cluster/apps/external-secrets/stores --server-side

tailscale:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/tailscale --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status deployment/operator -n core-tailscale --timeout=120s

argocd:
	kubectl apply -k cluster/namespaces --server-side
	kubectl kustomize cluster/apps/argocd --enable-helm | kubectl apply --server-side -f -
	kubectl rollout status deployment/argocd-server -n core-argocd --timeout=120s