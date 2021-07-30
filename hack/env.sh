#!/usr/bin/env bash

# Get info of docker image
COMPONENT_NAME="antrea"

# Get info of chart
CHART_NAME="antrea"
CHART_VERSION="v1.5.0"

IMG_VERSION="master"
TOS_IMAGE_REPO="tostmp"

if [[ "$OSTYPE" == "darwin"* ]]; then
  echo "Mac OS Darwin" && SED=gsed
  if ! type gsed >/dev/null 2>&1; then
    echo "Missing tool of gnu-sed for this script." && exit 1
  fi
else
  echo "Linux-GNU" && SED=sed
fi

set +e
# Check if the tree is dirty
if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
  GIT_TREE_STATE="clean"
else
  GIT_TREE_STATE="dirty"
fi

# Get the branch name
BRANCH_NAME="$(git symbolic-ref --short -q HEAD)"
if [[ -n "$BRANCH_NAME" ]]; then
  IMG_VERSION="$BRANCH_NAME"
fi

CHECK_TAGS=$(git tag)
# check if exist tag, use `git describe --tags` command will have can not describe anything error, if there has no tag.
if [[ -n "$CHECK_TAGS" ]];then
  # Get the latest full tag and judge the tag name whether belongs to rc
  TAGNAME_FULL=$(git describe --tags)
  NAME_RC=$(echo "$TAGNAME_FULL" | grep -E "rc|beta|alpha")

  # Get the latest short tag and get the hash of this tag
  TAGNAME_LATEST=$(git describe --abbrev=0 --tags)
  TAG_HASH=$(git rev-list -n 1 "${TAGNAME_LATEST}")

  # Get the latest commit hash
  HEAD_HASH=$(git rev-parse HEAD)
fi
if [ ${GIT_TREE_STATE} == "clean" ] && [ -z "${NAME_RC}" ] && [ -n "${TAG_HASH}" ] && [ "$TAG_HASH" == "$HEAD_HASH" ]; then
  # Check if the latest tag is on the lastest commit
  TOS_IMAGE_REPO="tosfinal"
  IMG_VERSION=$TAGNAME_LATEST
else
  if [ -n "${TAG_HASH}" ] && [ "$TAG_HASH"x == "$HEAD_HASH"x ]; then
    IMG_VERSION=$TAGNAME_LATEST
  fi
fi


# Other dirs
DEPLOY_DIR="build/deploy"
CHARTS_DIR="build/charts"
YAMLS_DIR="${DEPLOY_DIR}/yamls"

# files list of which need to be replaced
FILES="${DEPLOY_DIR}/images.yaml ${DEPLOY_DIR}/yamls/antrea-values.yaml.tmpl ${CHARTS_DIR}/${CHART_NAME}/values.yaml ${CHARTS_DIR}/${CHART_NAME}/Chart.yaml"

export GOPROXY=http://172.26.0.104:3000,https://goproxy.io,https://goproxy.cn,direct

set -e
