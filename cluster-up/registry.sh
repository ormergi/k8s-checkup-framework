#!/usr/bin/env bash

set -e

readonly HOST_PORT="5000"
readonly REGISTRY_CONTAINER_NAME="registry"
readonly REGISTRY_IMAGE="registry:2"

CRI=${CRI:-podman}

function cri() {
  $CRI "$@"
}

function registry::run_registry() {
    local -r network=${1}

    while cri inspect $REGISTRY_CONTAINER_NAME >> /dev/null; do
        cri stop $REGISTRY_CONTAINER_NAME || true
        cri rm   $REGISTRY_CONTAINER_NAME || true
        sleep 10
    done
    cri  run -d --network=${network} -p $HOST_PORT:5000  --restart=always --name $REGISTRY_CONTAINER_NAME registry:2
}

function registry::configure_registry_on_node() {
    local -r node=${1}
    local -r network=${2}

    _configure-insecure-registry-and-reload "cri exec -it -d ${node} bash -c"
}

function _configure-insecure-registry-and-reload() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "$(_insecure-registry-config-cmd)"
    ${cmd_context} "$(_reload-containerd-daemon-cmd)"
    ${cmd_context} "$(_add-registry-to-hosts-file)"
}
function _reload-containerd-daemon-cmd() {
    echo "systemctl restart containerd"
}

function _insecure-registry-config-cmd() {
    echo "sed -i '/\[plugins.cri.registry.mirrors\]/a\        [plugins.cri.registry.mirrors.\"registry:5000\"]\n\          endpoint = [\"http://registry:5000\"]' /etc/containerd/config.toml"
}
function _add-registry-to-hosts-file() {
    registry_container_ip=$(cri inspect $REGISTRY_CONTAINER_NAME --format "{{.NetworkSettings.Networks.${network}.IPAddress}}")
    echo "printf \"$registry_container_ip\tregistry\n\" >> /etc/hosts"
}