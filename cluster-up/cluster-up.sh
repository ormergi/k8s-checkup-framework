#!/usr/bin/env bash

set -e

SCRIPT_PATH="$(dirname "$(realpath "$0")")"

CRI=${CRI:-podman}

readonly CLUSTER_DIR="_cluster"
readonly KIND="$CLUSTER_DIR/.kind"
readonly KUBECTL="$CLUSTER_DIR/.kubectl"
readonly KUBECONFIG="$CLUSTER_DIR/.kubeconfig"

readonly KIND_VERSION="0.11.1"
readonly ARCH="amd64"

readonly KIND_NODE_KUBECTL_PATH="/bin/kubectl"
readonly KIND_DEFAULT_NETWORK="kind"

readonly REGISTRY_LIB="$SCRIPT_PATH/registry.sh"


function cri() {
  $CRI "$@"
}

function kind() {
  $KIND "$@"
}

function kubectl() {
  $KUBECTL "$@"
}

function fetch_kind() {
    current_kind_version=$(kind --version | awk '{print $3}')
    if [[ $current_kind_version != $KIND_VERSION ]]; then
        echo "Downloading kind v${KIND_VERSION}"
        curl -L "https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-${ARCH}" -o "$KIND"
        chmod +x "$KIND"
    fi
}

function spin_up_kind_cluster() {
    if [[ -z "$(kind get clusters | grep $CLUSTER_NAME)" ]]; then
      # create kind cluster
      kind create cluster --retain --name="$CLUSTER_NAME" --config="$CLUSTER_CONFIG" --image="$KIND_NODE_IMAGE" --verbosity=10
    fi

    # fetch kubeconfig from cluster
    kind get kubeconfig --name="$CLUSTER_NAME" > "$KUBECONFIG"
    export KUBECONFIG
    # fetch kubectl from cluster
    cri cp "$CLUSTER_NAME-control-plane":"$KIND_NODE_KUBECTL_PATH" "$KUBECTL"
    chmod u+x "$KUBECTL"
    # wait for cluster to be ready
    until kubectl get nodes; do echo "Waiting for nodes to become ready ..."; sleep 10; done
    kubectl wait nodes --all --for=condition=ready --timeout 15m
    kubectl wait pods -n kube-system --all --for=condition=Ready --timeout 15m

    kubectl cluster-info
}

mkdir -p "$CLUSTER_DIR"

fetch_kind

readonly CLUSTER_NAME="test"
readonly KIND_NODE_IMAGE="kindest/node:v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6"
readonly CLUSTER_CONFIG="cluster-up/manifests/kind.yaml"
spin_up_kind_cluster

# deploy local registry
source $REGISTRY_LIB
registry::run_registry "$KIND_DEFAULT_NETWORK"
# setup cluster nodes to work with the local registry
for node in $(kubectl get nodes --no-headers | awk '{print $1}'); do
    registry::configure_registry_on_node "$node" "$KIND_DEFAULT_NETWORK"
done

kubectl get nodes
kubectl get pods -A -o wide
echo ""
echo "cluster '$CLUSTER_NAME' is ready"