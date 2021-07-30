#!/bin/bash

source hack/env.sh

# Use helm v3 to package the chart to .tgz
cd ${CHARTS_DIR}/${CHART_NAME}
helm package . && TGZNAME=$(ls . | grep .tgz)
curl --data-binary @./${TGZNAME} -u chart_controller:cvlPxrJq1QfUmLTB http://172.16.1.99:9999/api/${TOS_IMAGE_REPO}/charts
