#!/bin/bash

source hack/env.sh

# replace the fields in images.yaml, Charts.yaml and values.yaml
for file in ${FILES}
do
  echo ${file}
  ${SED} -i "s/{{ TOS_IMAGE_REPO }}/${TOS_IMAGE_REPO}/g"  ${file}
  ${SED} -i "s/{{ IMG_VERSION }}/${IMG_VERSION}/g" ${file}
  ${SED} -i "s/{{ CHART_VERSION }}/${CHART_VERSION}/g" ${file}
done
