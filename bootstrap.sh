#!/usr/bin/env bash
set -euo pipefail

kubectl apply --server-side --kustomize k8s/core/external-secrets/crds

kubectl apply --kustomize k8s/namespaces

kubectl apply --kustomize k8s/bootstrap

kubectl apply --kustomize k8s/core/external-secrets/app

kubectl apply --kustomize k8s/core/cilium/app

kubectl apply --server-side --kustomize k8s/core/argocd/crds

kubectl apply --kustomize k8s/core/argocd/app
