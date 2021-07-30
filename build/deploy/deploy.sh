#!/usr/bin/env bash
set -x

SHELL_FOLDER=$(dirname $(readlink -f "$0"))
echo "SHELL_FOLDER $SHELL_FOLDER"

function deploy_antrea {
  envtpl -m error yamls/antrea-values.yaml.tmpl > yamls/antrea-values.yaml

  helmv3 -n "${NAMESPACE}" list | grep antrea | grep failed
  if [ $? -eq 0 ]; then
    helmv3 -n "${NAMESPACE}" delete antrea
  fi
  TGZ_NAME=$(ls "${SHELL_FOLDER}"/charts | grep .tgz)

  helmv3 upgrade -i -n "${NAMESPACE}" -f "${SHELL_FOLDER}"/yamls/antrea-values.yaml antrea "${SHELL_FOLDER}"/charts/"${TGZ_NAME}"
}

source /etc/profile
export NAMESPACE=${NAMESPACE:-"kube-system"}

export KUBE_APISERVER=$(kubectl get po -n kube-system | grep apiserver | awk 'NR==1 {print $1}')
export SERVICECIDR=$(kubectl get po ${KUBE_APISERVER} -n kube-system -oyaml | grep service-cluster-ip-range | tr ' ' '\n' | grep service-cluster-ip-range | awk -F "=" '{print $2}')


deploy_antrea
