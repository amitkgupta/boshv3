#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

kubectl delete deployment.bosh.akgupta.ca --all --namespace=test
kubectl delete extension,baseimage,role.bosh.akgupta.ca,release,network,az --all --namespace=test
kubectl delete team --all --namespace=test

kubectl delete compilation --all --namespace=bosh-system
kubectl delete director --all --namespace=bosh-system

kubectl delete secret vbox-admin-client-secret --namespace=bosh-system

kubectl delete namespace test