#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="gitops-study"
KIND_CONFIG="$(dirname "$0")/../cluster/kind-config.yaml"

# Check if kind is installed
if ! command -v kind &> /dev/null; then
  echo "Error: kind is not installed."
  echo "Install it from https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
  exit 1
fi

# Delete existing cluster if it exists
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "Deleting existing kind cluster '${CLUSTER_NAME}'..."
  kind delete cluster --name "${CLUSTER_NAME}"
fi

# Create cluster
echo "Creating kind cluster '${CLUSTER_NAME}'..."
kind create cluster --name "${CLUSTER_NAME}" --config "${KIND_CONFIG}"

# Wait for nodes to be ready
echo "Waiting for nodes to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s

# Install metrics-server
echo "Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Patch metrics-server for kind (disable TLS verification)
kubectl patch deployment metrics-server -n kube-system \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'

# Wait for metrics-server to be ready
echo "Waiting for metrics-server to be ready..."
kubectl wait --for=condition=Available deployment/metrics-server -n kube-system --timeout=120s

# Print cluster info
echo ""
echo "============================================"
echo "Kind cluster '${CLUSTER_NAME}' is ready!"
echo "============================================"
kubectl cluster-info
echo ""
kubectl get nodes -o wide
