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
    if minikube status -p parca-agent; then
        echo "----------------------------------------------------------"
        echo "Dev cluster already running. Skipping minikube cluster creation"
        echo "----------------------------------------------------------"
    else
        ctlptl create registry ctlptl-registry || echo 'Registry already exists'
        minikube start -p parca-agent --driver=virtualbox --kubernetes-version=v1.22.0 --cpus=4 --disk-size=40000mb
    fi

    # Pull parca-agent repo to build live image
    if [ -d "tmp/parca-agent" ]
    then
        pushd tmp/parca-agent
        git pull origin main
        git submodule init && git submodule update
        popd
    else
        git clone git@github.com:parca-dev/parca-agent.git tmp/parca-agent
        pushd tmp/parca-agent
        git submodule init && git submodule update
        popd
    fi

    # Deploy all services into the cluster
    deploy

    echo "Now run \"tilt up\" to start developing!"
}

# Tears down a local minikube cluster
function down() {
    minikube delete -p parca-agent
    rm -rf tmp/parca-agent
}

# Deploys the dev env into the minikube cluster
function deploy() {
    # Deploy all generated manifests
    kubectl apply -R -f ./deploy/manifests
}
