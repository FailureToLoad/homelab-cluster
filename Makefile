.PHONY: bootstrap talosconfigs flux-bootstrap flux-reconcile flux-status pre-secrets

bootstrap:
	cd bootstrapper && go run .

talosconfigs:
	cd configmaker && go run .

pre-secrets:
ifndef BWS_ACCESS_TOKEN
	$(error BWS_ACCESS_TOKEN is not set)
endif
ifndef BWS_ORG_ID
	$(error BWS_ORG_ID is not set)
endif
ifndef BWS_PROJECT_ID
	$(error BWS_PROJECT_ID is not set)
endif
	@echo "Creating flux-system namespace..."
	kubectl create namespace flux-system --dry-run=client -o yaml | kubectl apply -f -
	@echo "Creating external-secrets namespace..."
	kubectl create namespace external-secrets --dry-run=client -o yaml | kubectl apply -f -
	@echo "Creating cluster-secrets for variable substitution..."
	kubectl create secret generic cluster-secrets \
		--namespace flux-system \
		--from-literal=BWS_ORG_ID="$$BWS_ORG_ID" \
		--from-literal=BWS_PROJECT_ID="$$BWS_PROJECT_ID" \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Creating bitwarden-access-token secret..."
	kubectl create secret generic bitwarden-access-token \
		--namespace external-secrets \
		--from-literal=token="$$BWS_ACCESS_TOKEN" \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Pre-bootstrap secrets created."

flux-bootstrap: pre-secrets
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN is not set)
endif
	flux bootstrap github \
		--owner=FailureToLoad \
		--repository=homelab-cluster \
		--branch=main \
		--path=./cluster/flux-system \
		--personal

flux-reconcile:
	flux reconcile source git flux-system
	flux reconcile kustomization flux-system

flux-status:
	@echo "=== Flux Sources ==="
	flux get sources all
	@echo ""
	@echo "=== Flux Kustomizations ==="
	flux get kustomizations
	@echo ""
	@echo "=== Helm Releases ==="
	flux get helmreleases -A
