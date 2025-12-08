.PHONY: bootstrap flux-reconcile flux-status

bootstrap:
	cd bootstrapper && go run .

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
