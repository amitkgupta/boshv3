#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl create namespace test

kubectl create secret generic vbox-admin-client-secret \
  --from-literal=secret="$(bosh interpolate --path /uaa_admin_client_secret ${HOME}/envs/vbox/creds.yml)" \
  --namespace=bosh-system

kubectl apply --filename="${__dir}/../../config/samples"