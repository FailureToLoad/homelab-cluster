.PHONY: bootstrap talosconfigs namespaces certmanager external-secrets tailscale

bootstrap:
	cd bootstrapper && go run .

talosconfigs:
	cd configmaker && go run .

namespaces:
	kubectl apply -k cluster/namespaces

certmanager:
	kubectl kustomize cluster/apps/cert-manager --enable-helm | kubectl apply -f -
	@echo "Waiting for cert-manager to be ready..."
	kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s
	kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s

	@echo "Annotating cilium-ca secret for cert-manager adoption..."
	kubectl annotate secret cilium-ca -n cilium \
		cert-manager.io/certificate-name=cilium-ca \
		cert-manager.io/issuer-kind=ClusterIssuer \
		cert-manager.io/issuer-name=cilium-selfsigned \
		--overwrite

	@echo "Annotating hubble-server-certs secret for cert-manager adoption..."
	kubectl annotate secret hubble-server-certs -n cilium \
		cert-manager.io/certificate-name=hubble-server-certs \
		cert-manager.io/issuer-kind=Issuer \
		cert-manager.io/issuer-name=cilium-ca-issuer \
		--overwrite

	@echo "Applying Cilium certificate resources..."
	kubectl apply -f cluster/apps/cilium/certificates.yaml
	@echo "cert-manager installed and managing Cilium certificates."

external-secrets:
ifndef BWS_ACCESS_TOKEN
	$(error BWS_ACCESS_TOKEN is not set)
endif
ifndef BWS_ORG_ID
	$(error BWS_ORG_ID is not set)
endif
ifndef BWS_PROJECT_ID
	$(error BWS_PROJECT_ID is not set)
endif
	kubectl kustomize cluster/apps/external-secrets --enable-helm | kubectl apply --server-side -f -

	@echo "Applying bitwarden-sdk-server certificate..."
	kubectl apply -f cluster/apps/external-secrets/certificates.yaml

	@echo "Waiting for external-secrets to be ready..."
	kubectl wait --for=condition=Available deployment/external-secrets -n external-secrets --timeout=120s
	kubectl wait --for=condition=Available deployment/external-secrets-webhook -n external-secrets --timeout=120s
	kubectl wait --for=condition=Available deployment/bitwarden-sdk-server -n external-secrets --timeout=120s
	kubectl wait --for=condition=Established crd/clustersecretstores.external-secrets.io --timeout=60s
	@sleep 3

	@echo "Creating Bitwarden access token secret..."
	kubectl create secret generic bitwarden-access-token \
		--namespace external-secrets \
		--from-literal=token="$$BWS_ACCESS_TOKEN" \
		--dry-run=client -o yaml | kubectl apply -f -

	@echo "Applying ClusterSecretStore..."
	BWS_ORG_ID="$$BWS_ORG_ID" BWS_PROJECT_ID="$$BWS_PROJECT_ID" \
		envsubst < cluster/apps/external-secrets/cluster-secret-store.yaml | kubectl apply -f -
	@echo "external-secrets installed with Bitwarden Secrets Manager."

tailscale: namespaces
	@echo "Waiting for tailscale oauth secret to be synced..."
	kubectl wait --for=condition=Ready externalsecret/tailscale-operator-oauth -n tailscale --timeout=120s

	kubectl kustomize cluster/apps/tailscale --enable-helm | kubectl apply --server-side -f -
	@echo "Waiting for tailscale-operator to be ready..."
	kubectl wait --for=condition=Available deployment/operator -n tailscale --timeout=120s
	@echo "Tailscale operator installed."
