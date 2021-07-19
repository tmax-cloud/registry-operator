#!/bin/bash
set -e

INGRESS_IP=$1
CERTIFICATE=$2
PRIVATE_KEY=$3

CreateCACertAndKey() {
  openssl req -x509 -nodes -days 3650 -newkey rsa:4096 \
        -keyout ca.key -out ca.crt \
        -subj "/C=KR/ST=Seoul/L=Seoul/O=Tmax"
}

Usage() {
    echo "Usage: $0 <ingress_ip> <cacert> <ca_key>"
}

if [[ -z $INGRESS_IP ]]; then
  echo "[Error]: No ingress IP specified."
  Usage
  exit
fi

if [[ -z $CERTIFICATE ]]; then
	CERTIFICATE=ca.crt
	PRIVATE_KEY=ca.key
	if [[ ! -e $CERTIFICATE ||  ! -e $PRIVATE_KEY ]]; then
	  echo "no CA certificate and key found. create new one..."
	  CreateCACertAndKey
	fi
fi

kubectl create secret generic registry-ca \
  -n registry-system \
  --from-file=ca.crt=${CERTIFICATE} \
  --from-file=ca.key=${PRIVATE_KEY} \

cat <<EOF > /tmp/token-service.conf
[ req ]
default_bits            = 2048
default_md              = sha1
default_keyfile         = ca.key
distinguished_name      = req_distinguished_name
extensions              = v3_user

[ v3_user ]
# Extensions to add to a certificate request
basicConstraints = CA:FALSE
authorityKeyIdentifier = keyid,issuer
subjectKeyIdentifier = hash
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth,clientAuth
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = auth.hyperregistry.__DOMAIN__

[ req_distinguished_name ]
countryName                     = Country Name (2 letter code)
countryName_default             = KR
countryName_min                 = 2
countryName_max                 = 2

organizationName              = Organization Name (eg, company)
organizationName_default      = TmaxCloud

organizationalUnitName          = Organizational Unit Name (eg, section)
organizationalUnitName_default  = DevOps

commonName                      = Common Name (eg, hostname)
commonName_default             = token-service
commonName_max                  = 64
EOF

sed -i "s/__DOMAIN__/${INGRESS_IP}.nip.io/" /tmp/token-service.conf
sed "s/__DOMAIN__/${INGRESS_IP}.nip.io/" ./config/token/ingress-tpl.yaml > ./config/token/ingress.yaml
sed "s/__DOMAIN__/${INGRESS_IP}.nip.io/" ./config/manager/manager_config-tpl.yaml > ./config/manager/manager_config.yaml

openssl req -new -key ${PRIVATE_KEY} -config /tmp/token-service.conf -out /tmp/token-service.csr
openssl x509 -req -days 1825 -in /tmp/token-service.csr -extensions v3_user -extfile /tmp/token-service.conf \
    -CA ${CERTIFICATE} -CAcreateserial \
    -CAkey ${PRIVATE_KEY} \
    -out /tmp/token-service.pem

kubectl create secret tls token-service \
  -n registry-system \
  --cert=/tmp/token-service.pem \
  --key=${PRIVATE_KEY}

kubectl apply -f config/manager/namespace.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -f config/rbac/image-signer-role.yaml
kubectl apply -f config/crd/bases
kubectl apply -f config/manager/configmap.yaml
kubectl apply -f config/manager/keycloak_secret.yaml
kubectl apply -f config/apiservice/apiservice.yaml
kubectl apply -f config/webhook/mutating-webhook.yaml
kubectl apply -f config/manager/manager_config.yaml

kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/manager/job_manager.yaml
kubectl apply -f config/manager/service.yaml

kubectl apply -f config/token/keycloak.yaml
kubectl apply -f config/token/postgresql.yaml
kubectl apply -f config/token/ingress.yaml
