#!/usr/bin/env bash
set -x

SHELL_FOLDER=$(dirname $(readlink -f "$0"))

function import_oci {
  echo "import oci $1"
  jq -r '.manifests[].annotations."org.opencontainers.image.ref.name"' ${1}/index.json | \
    xargs -i skopeo --insecure-policy=true copy oci:${1}:{} docker://${REGISTRY_ADDR}/{}
}

# ARGS ENV PARAMS
export REGISTRY_ADDR=${REGISTRY_ADDR:-"127.0.0.1:5000"}

import_oci ${SHELL_FOLDER}/images
