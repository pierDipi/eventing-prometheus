#!/usr/bin/env bash

set -euo pipefail

source $(dirname $0)/../vendor/knative.dev/hack/library.sh

repo_root_dir=$(dirname "$(realpath "${BASH_SOURCE[0]}")")/..

kubectl apply --server-side -f "${repo_root_dir}/third_party/prometheus-operator/kube-prometheus/manifests/setup"
# Wait until the "servicemonitors" CRD is created. The message "No resources found" means success in this context.
until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
kubectl apply --server-side -f "${repo_root_dir}/third_party/prometheus-operator/kube-prometheus/manifests"

kubectl patch alertmanager --type=merge -n monitoring main --patch-file "${repo_root_dir}/third_party/prometheus-operator/alertmanager-patch.yaml"

wait_until_pods_running monitoring || fail_test "Failed to start up kube-prometheus"
