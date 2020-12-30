#!/bin/bash

set -e

# Create registry-system namespace
kubectl apply -f config/manager/namespace.yaml

# Apply role
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml

# Apply CRDs
kubectl apply -f config/crd/bases

# Apply confimap
kubectl apply -f config/manager/configmap.yaml

# Apply keycloak secret
kubectl apply -f config/manager/keycloak_secret.yaml

# Apply
kubectl apply -f config/apiservice/apiservice.yaml

# Create registry CA
CA_CRT_FILE=./config/pki/ca.crt
CA_KEY_FILE=./config/pki/ca.key
if [ ! -f "$CA_CRT_FILE" ] || [ ! -f "$CA_KEY_FILE" ]; then
    echo "$CA_CRT_FILE is not exist... Use default CA"
    echo "Warning!!! Please run the 'config/scripts/newCertFile.sh' to create new root ca file. Using default CA is vulnerable."
    CA_CRT_FILE=./config/pki/default_ca.crt
    CA_KEY_FILE=./config/pki/default_ca.key
fi

. ./config/scripts/newCertSecret.sh $CA_CRT_FILE $CA_KEY_FILE

# Deploy operator
kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/manager/service.yaml

echo "deploy registry-operator success"

exit 0