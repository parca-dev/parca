#!/usr/bin/env bash

################################################################################
#
# This script is meant to be run from the root of this project with the Makefile
#
################################################################################

set -euo pipefail

# Creates a local minikube cluster, and deploys the dev env into the cluster
function up() {
    # Spin up local cluster if one isn't running
    if minikube status -p parca; then
        echo "----------------------------------------------------------"
        echo "Dev cluster already running. Skipping minikube cluster creation"
        echo "----------------------------------------------------------"
    else
        ctlptl create registry ctlptl-registry || echo 'Registry already exists'
        minikube start -p parca --driver=virtualbox --kubernetes-version=v1.22.3 --cpus=4 --disk-size=80000mb --docker-opt dns=8.8.8.8
        # minikube start -p parca --driver=kvm2 --kubernetes-version=v1.22.3 --cpus=12 --memory=40gb --disk-size=120gb --docker-opt dns=8.8.8.8
    fi

    # Deploy all services into the cluster
    deploy

    echo "Now run \"tilt up\" to start developing!"
}

# Tears down a local minikube cluster
function down() {
    minikube delete -p parca
}

# Deploys the dev env into the minikube cluster
function deploy() {
    # Deploy all generated manifests
    kubectl apply -R -f ./deploy/tilt
}
