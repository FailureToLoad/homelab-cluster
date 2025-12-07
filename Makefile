.PHONY: bootstrap talosconfigs certmanager

bootstrap:
	cd bootstrapper && go run .

talosconfigs:
	cd configmaker && go run .

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
