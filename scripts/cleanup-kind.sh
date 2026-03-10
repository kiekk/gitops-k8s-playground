#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="gitops-study"

if ! command -v kind &> /dev/null; then
  echo "Error: kind is not installed."
  exit 1
fi

if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "Deleting kind cluster '${CLUSTER_NAME}'..."
  kind delete cluster --name "${CLUSTER_NAME}"
  echo "Cluster '${CLUSTER_NAME}' deleted."
else
  echo "Cluster '${CLUSTER_NAME}' does not exist."
fi
