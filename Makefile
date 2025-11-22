.PHONY: vaultmaker
vaultmaker:
	cd homelabtools && go run ./cmd/vaultmaker/main.go

.PHONY: generate
generate:
	cd homelabtools && go run ./cmd/generate/main.go

.PHONY: generate-overwrite
generate-overwrite:
	cd homelabtools && go run ./cmd/generate/main.go -overwrite

.PHONY: fetch-secrets
fetch-secrets:
	mkdir -p k8s/bootstrap/secrets/cilium
	mkdir -p k8s/bootstrap/secrets/azure
	SECRETS_PATH=$$(realpath ./k8s/bootstrap/secrets) && cd homelabtools && go run ./cmd/customresourcevalues $$SECRETS_PATH

.PHONY: bootstrap
bootstrap:
	kubesource
	$(MAKE) fetch-secrets
	./bootstrap.sh