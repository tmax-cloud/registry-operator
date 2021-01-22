#!/bin/bash

set -e

# Create registry-system namespace
kubectl apply -f config/manager/namespace.yaml

# Apply role
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -f config/rbac/image-signer-role.yaml

# Apply CRDs
kubectl apply -f config/crd/bases

# Apply confimap
kubectl apply -f config/manager/configmap.yaml

# Apply keycloak secret
kubectl apply -f config/manager/keycloak_secret.yaml

# Apply apiservice
kubectl apply -f config/apiservice/apiservice.yaml

# Apply webhook
kubectl apply -f config/webhook/mutating-webhook.yaml

# Apply manager config
kubectl create configmap manager-config -n registry-system --from-file=config/manager/manager_config.yaml || true

# Create registry CA
CA_CRT_FILE=./config/pki/ca.crt
CA_KEY_FILE=./config/pki/ca.key
if [ ! -f "$CA_CRT_FILE" ] || [ ! -f "$CA_KEY_FILE" ]; then
    echo "$CA_CRT_FILE is not exist... Use default CA"
    echo "Warning!!! Please run the 'config/scripts/newCertFile.sh' to create new root ca file. Using default CA is vulnerable."
    CA_CRT_FILE=./config/pki/default_ca.crt
    CA_KEY_FILE=./config/pki/default_ca.key
fi

# Create registry-ca secret
. ./config/scripts/newCertSecret.sh registry-ca $CA_CRT_FILE $CA_KEY_FILE

# Create keycloak-cert secret
KEYCLOAK_CRT_FILE=./config/pki/keycloak.crt
if [[ -f "$KEYCLOAK_CRT_FILE" ]]; then 
    . ./config/scripts/newCertSecret.sh keycloak-cert $KEYCLOAK_CRT_FILE
fi

# Deploy operator
kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/manager/service.yaml

echo "deploy registry-operator success"

exit 0