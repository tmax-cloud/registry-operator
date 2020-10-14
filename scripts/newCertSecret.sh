#/bin/sh

CA_CRT=./ca.crt
CA_KEY=./ca.key

if [ -n "$1" ] && [ -n "$2" ]; then
    CA_CRT=$1
    CA_KEY=$2
fi

echo "Create CA($CA_CRT, $CA_KEY) registry-ca secret"
kubectl create secret generic registry-ca --from-file=ca.crt=${CA_CRT} --from-file=ca.key=${CA_KEY} -n registry-system