#!/bin/bash
set -x

WORKDIR=$(cd $(dirname $0);pwd)
export ARCH="arm64"
export PROJECT_NAME="antrea"
export IMAGE_YAML="${WORKDIR}/images-arm64.yaml"
 
bash -x ${WORKDIR}/render.sh
 
bash -x ${WORKDIR}/../../package-service/package-util.sh