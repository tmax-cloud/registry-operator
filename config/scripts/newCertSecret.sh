#/usr/bin/env bash

set -e

Usage() {
    echo "Usage: $0 <secret_name> [<crt_file_path> [<key_file_path>]]"
}

if ! ( [[ "$#" = 1 ]] || [[ "$#" = 2 ]] || [[ "$#" = 3 ]] ); then 
    Usage
    exit
fi

CA_CRT=$2
CA_KEY=$3

if [[ -z $CA_CRT ]]; then
	CA_CRT=./ca.crt
	CA_KEY=./ca.key
fi

if [[ -z $CA_KEY ]]; then
	echo "Create CA($CA_CRT) $1 secret"
	kubectl create secret generic $1 --from-file=ca.crt=${CA_CRT} -n registry-system || true
else
	echo "Create CA($CA_CRT, $CA_KEY) $1 secret"
	kubectl create secret generic $1 --from-file=ca.crt=${CA_CRT} --from-file=ca.key=${CA_KEY} -n registry-system || true
fi

echo "Create CA Completed"

