#! /bin/bash

set -e 

SCRIPT_PATH=$(dirname "$(realpath "$0")")
MANIFESTS="$SCRIPT_PATH/manifests"

CRI="${CRI:-podman}"

# build kubevirt latency checkup container
IMAGE="kubevirt-latency-checkup"
TAG="latest"
CRI="$CRI" IMAGE="$IMAGE" TAG="$TAG" $SCRIPT_PATH/build/build-image

# push to local registry
REGISTRY="${REGISTRY:-localhost:5000}"
$CRI tag "${IMAGE}:${TAG}" "${REGISTRY}/${IMAGE}:${TAG}"
$CRI push "${REGISTRY}/${IMAGE}:${TAG}"

trap "kubectl delete -f $MANIFESTS/standalone" EXIT

## Create kubevirt network latency checkup prequisits
kubectl apply -f $MANIFESTS/nads.yaml

kubectl apply -f $MANIFESTS/standalone/namespace.yaml
kubectl apply -f $MANIFESTS/standalone/serviceaccount.yaml
kubectl apply -f $MANIFESTS/standalone/roles.yaml
kubectl apply -f $MANIFESTS/standalone/rolebindings.yaml
kubectl apply -f $MANIFESTS/standalone/results-configmap.yaml
kubectl apply -f $MANIFESTS/standalone/latency-check-job.yaml

# follow the checkup logs..
working_ns=$(cat $MANIFESTS/standalone/namespace.yaml | grep -Po "name: \K.*")
checkup_job=$(cat $MANIFESTS/standalone/latency-check-job.yaml | grep metadata: -A 2 | grep -Po "name: \K.*")
job_name_label="job-name=$checkup_job"

kubectl get job $checkup_job -n $working_ns
echo "waiting for job pod to start.."
timeout 30s bash -c "until kubectl get pod -n $working_ns -l $job_name_label --field-selector status.phase=Running --no-headers | grep . ; do sleep 2; done" || true

pod=$(kubectl get po -n $working_ns -l $job_name_label --no-headers | head -1 | awk '{print $1}')
kubectl logs $pod -n $working_ns --follow | tee

kubectl wait job -n $working_ns $checkup_job --for condition=complete

# print latency check results from the result ConfigMap
results_configmap=$(cat $MANIFESTS/standalone/results-configmap.yaml | grep -Po "name: \K.*")
results_configmap_ns=$(cat $MANIFESTS/standalone/results-configmap.yaml | grep -Po "namespace: \K.*")
kubectl get cm $results_configmap -n $results_configmap_ns -o jsonpath='{.data}' | jq
