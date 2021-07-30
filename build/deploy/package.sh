#!/usr/bin/env bash
set -x
WORKDIR=$(cd $(dirname $0);pwd)
export ARCH="x86_64"
export PROJECT_NAME="antrea"
export IMAGE_YAML="${WORKDIR}/images.yaml"

bash -x ${WORKDIR}/../../hack/render.sh

# 如果需要渲染项目的images.yaml以及charts中的values.yaml中的version或者tag信息，请在此处执行自己项目的渲染脚本进行渲染，具体实现不做限制
bash -x "${WORKDIR}"/../../package-service/package-util.sh
