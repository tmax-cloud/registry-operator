#!/bin/bash

set -e

# Create registry-system namespace
kubectl apply -f deploy/namespace.yaml

# Apply role
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/cluster_role.yaml
kubectl apply -f deploy/cluster_role_binding.yaml

# Apply CRDs
kubectl apply -f deploy/crds/tmax.io_registries_crd.yaml
kubectl apply -f deploy/crds/tmax.io_repositories_crd.yaml

# Apply confimap
kubectl apply -f deploy/configmap.yaml

# Create registry CA
CA_CRT_FILE=./deploy/pki/ca.crt
CA_KEY_FILE=./deploy/pki/ca.key
if [ ! -f "$CA_CRT_FILE" ] || [ ! -f "$CA_KEY_FILE" ]; then
    echo "$CA_CRT_FILE is not exist... Use default CA"
    echo "Warning!!! Please run the 'scripts/newCertFile.sh' to create new root ca file. Using default CA is vulnerable."
    CA_CRT_FILE=./deploy/pki/default_ca.crt
    CA_KEY_FILE=./deploy/pki/default_ca.key
fi

. ./scripts/newCertSecret.sh $CA_CRT_FILE $CA_KEY_FILE

# Deploy operator
kubectl apply -f deploy/operator.yaml
kubectl apply -f deploy/operator_service.yaml

echo "deploy registry-operator success"

exit 0