.PHONY: vaultmaker
vaultmaker:
	cd homelabtools && go run ./cmd/vaultmaker/main.go

.PHONY: generate
generate:
	cd homelabtools && go run ./cmd/generate/main.go

.PHONY: generate-overwrite
generate-overwrite:
	cd homelabtools && go run ./cmd/generate/main.go -overwrite