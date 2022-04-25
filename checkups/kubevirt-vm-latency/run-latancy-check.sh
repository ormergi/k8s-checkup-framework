#! /bin/bash

set -e 

SCRIPT_PATH=$(dirname "$(realpath "$0")")
MANIFESTS="$SCRIPT_PATH/manifests"

CRI="${CRI:-podman}"

KUBECTL="${KUBECTL:-kubectl}"

KIND="${KIND:-./kind}"
CLUSTER_NAME="${CLUSTER_NAME:-sriov}"

# build kubevirt latency checkup container
IMAGE="kubevirt-vm-latency-checkup"
TAG="devel"
CRI="$CRI" IMAGE="$IMAGE" TAG="$TAG" $SCRIPT_PATH/build/build-image

# push image to nodes local registry
$KIND load docker-image "${IMAGE}:${TAG}" --name $CLUSTER_NAME 

trap "$KUBECTL delete -f $MANIFESTS/standalone 1> /dev/null" EXIT

# deploy kubevirt-vm-latency checkup
$KUBECTL apply -f $MANIFESTS/net-attach-defs.yaml
$KUBECTL apply -f $MANIFESTS/standalone

# follow the checkup logs..
working_ns=$(cat $MANIFESTS/standalone/0_namespace.yaml | grep -Po "name: \K.*")
checkup_job=$(cat $MANIFESTS/standalone/5_job.yaml | grep metadata: -A 2 | grep -Po "name: \K.*")
job_name_label="job-name=$checkup_job"

$KUBECTL get job $checkup_job -n $working_ns
echo "waiting for job pod to start.."
timeout 30s bash -c "until $KUBECTL get pod -n $working_ns -l $job_name_label --field-selector status.phase=Running --no-headers | grep . ; do sleep 2; done" || true

$KUBECTL logs job/$checkup_job -n $working_ns --follow | tee

while true; do 
    $KUBECTL wait job -n $working_ns $checkup_job --for condition=complete && break
    $KUBECTL wait job -n $working_ns $checkup_job --for condition=falied && break
    sleep 5
done

# print latency check results from the result ConfigMap
results_configmap=$(cat $MANIFESTS/standalone/2_results-configmap.yaml | grep -Po "name: \K.*")
results_configmap_ns=$(cat $MANIFESTS/standalone/2_results-configmap.yaml | grep -Po "namespace: \K.*")
$KUBECTL get cm $results_configmap -n $results_configmap_ns -o jsonpath='{.data}' | jq
