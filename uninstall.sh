#!/bin/bash

set -e

# Print usage
Usage() {
    echo "Usage:"
    echo "  $0 [OPTION]..." 
    echo

    echo "Uninstall Options"
    echo "  -a, --all"
    echo "                 delete all registry-operator resources"
    echo "  -c, --crd"
    echo "                 delete crds only"
    echo "  -m, --manager"
    echo "                 delete manager resources without crd resources"
    echo
    echo "  -s, --clair"
    echo "                 delete clair only"
    echo
}

# Delete manager resources without crd resources
DeleteManager() {
    # Delete operator
    kubectl delete -f config/manager/service.yaml
    kubectl delete -f config/manager/manager.yaml

    # Delete ca secret
    kubectl delete secret registry-ca -n registry-system

    # Delete manger config
    kubectl delete configmap manager-config -n registry-system

    # Delete webhook
    kubectl delete -f config/webhook/mutating-webhook.yaml

    # Delete apiservice
    kubectl delete -f config/apiservice/apiservice.yaml

    # Delete keycloak secret
    kubectl delete -f config/manager/keycloak_secret.yaml

    # Delete confimap
    kubectl delete -f config/manager/configmap.yaml

    # Delete role
    kubectl delete -f config/rbac/image-signer-role.yaml
    kubectl delete -f config/rbac/role_binding.yaml
    kubectl delete -f config/rbac/role.yaml

    # Delete registry-system namespace
    kubectl delete -f config/manager/namespace.yaml
}

# Delete crds only
DeleteCRD() {
    # Delete CRDs
    kubectl delete -f config/crd/bases
}

DeleteClair() {
    # Delete CRDs
    if [ -e config/manager/clair/dev.yml ]
    then
        kubectl delete -n registry-system -f config/manager/clair/dev.yml
        rm config/manager/clair/dev.yml
    else
        echo "Not found clair deployment file."
    fi
}


####################
##   Main Start   ##
####################

if [[ "$#" == 0 ]]; then
    Usage
    exit 0
fi

if ! options=$(getopt -o achms -l all,crd,help,manager,clair -- "$@")
then
    echo "ERROR: invalid command option"
    Usage
    exit 1
fi

eval set -- "$options"

while true; do
    case "$1" in
        -a|--all) 
            echo "* Tear down registry-operator"
            DeleteClair
            DeleteManager
            DeleteCRD
            shift ;;
        -c|--crd) 
            echo "* Tear down registry-operator crds"
            DeleteCRD
            shift ;;
        -h|--help) 
            Usage
            break
            ;;
        -m|--manager) 
            echo "* Tear down registry-operator resources without crds"
            DeleteManager
            shift ;;
        -s|--clair) 
            echo "* Tear down clair server"
            DeleteClair
            shift ;;
        --)           
            shift 
            break
            ;;
    esac
done

