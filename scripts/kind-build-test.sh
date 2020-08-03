#!/usr/bin/env bash

# A script that builds a single ACK service controller, provisions a KinD
# Kubernetes cluster, installs the built ACK service controller into that
# Kubernetes cluster and runs a set of tests

set -Eo pipefail

SCRIPTS_DIR=$(cd "$(dirname "$0")"; pwd)
ROOT_DIR="$SCRIPTS_DIR/.."

source "$SCRIPTS_DIR/lib/common.sh"

OPTIND=1
CLUSTER_NAME_BASE="test"
DELETE_CLUSTER_ARGS=""
K8S_VERSION="1.16"
OVERRIDE_PATH=0
PRESERVE=false
PROVISION_CLUSTER_ARGS=""
START=$(date +%s)
TMP_DIR=""
# VERSION is the source revision that executables and images are built from.
VERSION=$(git describe --tags --always --dirty || echo "unknown")

function timeout() { perl -e 'alarm shift; exec @ARGV' "$@"; }

function relpath() {
  perl -e 'use File::Spec; print File::Spec->abs2rel(@ARGV) . "\n"' "${1}" "${2}"
}

function clean_up {
    if [[ "$PRESERVE" == false ]]; then
        "${SCRIPTS_DIR}"/delete-kind-cluster.sh -c "$TMP_DIR" || :
        return
    fi
    echo "To resume test with the same cluster use: \"-c $TMP_DIR\""""
}

function exit_and_fail {
    END=$(date +%s)
    echo "⏰ Took $(expr "${END}" - "${START}")sec"
    echo "❌ ACK Integration Test FAILED $CLUSTER_NAME! ❌"
    exit 1
}

USAGE="
Usage:
  $(basename "$0") [-p] [-s] [-o] [-b <TEST_BASE_NAME>] [-c <CLUSTER_CONTEXT_DIR>] [-i <AWS Docker image name>] [-s] [-v K8S_VERSION]

Builds the Docker image for an ACK service controller, loads the Docker image
into a KinD Kubernetes cluster, creates the Deployment artifact for the ACK
service controller and executes a set of tests.

Example: $(basename "$0") -p -s ecr

Options:
  -b          Base name of test (will be used for cluster too)
  -c          Cluster context directory, if operating on an existing cluster
  -p          Preserve kind k8s cluster for inspection
  -i          Provide AWS Service docker image
  -s          Provide AWS Service name (ecr, sns, sqs, petstore, bookstore)
  -v          Kubernetes Version (Default: 1.16) [1.14, 1.15, 1.16, 1.17, and 1.18]
"

# Process our input arguments
while getopts "ps:ioc:b:v:" opt; do
  case ${opt} in
    p ) # PRESERVE K8s Cluster
        echo "❄️  This run will preserve the cluster as you requested"
        PRESERVE=true
      ;;
    s ) # AWS Service name
        AWS_SERVICE=$(echo "${OPTARG}" | tr '[:upper:]' '[:lower:]')
      ;;
    i ) # AWS Service Docker Image
        AWS_SERVICE_DOCKER_IMG="${OPTARG}"
      ;;
    c ) # Cluster context directory to operate on existing cluster
        TMP_DIR="${OPTARG}"
      ;;
    b ) # Base cluster name
        CLUSTER_NAME_BASE="${OPTARG}"
      ;;
    v ) # K8s VERSION
        K8S_VERSION="${OPTARG}"
      ;;
    \? )
        echo "${USAGE}" 1>&2
        exit
      ;;
  esac
done

if [ -z $TMP_DIR ]; then
    TMP_DIR=$("${SCRIPTS_DIR}"/provision-kind-cluster.sh -b "${CLUSTER_NAME_BASE}" -v "${K8S_VERSION}")
fi

if [ $OVERRIDE_PATH == 0 ]; then
  export PATH=$TMP_DIR:$PATH
else
  export PATH=$PATH:$TMP_DIR
fi

CLUSTER_NAME=$(cat $TMP_DIR/clustername)

## Build and Load Docker Images

if [ -z "$AWS_SERVICE_DOCKER_IMG" ]; then
    echo "🥑 Building ${AWS_SERVICE} docker image"
    DEFAULT_AWS_SERVICE_DOCKER_IMG="${AWS_SERVICE}:${VERSION}"
    docker build -f ${ROOT_DIR}/services/${AWS_SERVICE}/Dockerfile -t "${DEFAULT_AWS_SERVICE_DOCKER_IMG}" .
    AWS_SERVICE_DOCKER_IMG="${DEFAULT_AWS_SERVICE_DOCKER_IMG}"
    echo "👍 Built the ${AWS_SERVICE} docker image"
else
    echo "🥑 Skipping building the ${AWS_SERVICE} docker image, since one was specified ${AWS_SERVICE_DOCKER_IMG}"
fi
echo "$AWS_SERVICE_DOCKER_IMG" > "${TMP_DIR}"/"${AWS_SERVICE}"_docker-img

echo "🥑 Loading the images into the cluster"
kind load docker-image --name "${CLUSTER_NAME}" --nodes="${CLUSTER_NAME}"-worker,"${CLUSTER_NAME}"-control-plane "${AWS_SERVICE_DOCKER_IMG}"
echo "👍 Loaded image(s) into the cluster"

export KUBECONFIG="${TMP_DIR}/kubeconfig"

trap "exit_and_fail" INT TERM ERR
trap "clean_up" EXIT

echo "======================================================================================================"
echo "To poke around your test manually:"
echo "export KUBECONFIG=$TMP_DIR/kubeconfig"
echo "export PATH=$TMP_DIR:\$PATH"
echo "kubectl get pods -A"
echo "======================================================================================================"

# TODO: export any necessary env vars and run tests
