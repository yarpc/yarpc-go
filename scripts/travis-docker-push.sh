#!/bin/bash

set -e

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

if [ -n "${NO_DOCKER_PUSH}" ]; then
  exit 0
fi

COMMIT="${TRAVIS_COMMIT::8}"
REPO="yarpc/yarpc-go"
BRANCH="$(if [ "${TRAVIS_PULL_REQUEST}" == "false" ]; then echo ${TRAVIS_BRANCH}; else echo ${TRAVIS_PULL_REQUEST_BRANCH}; fi)"
TAG="$(if [ "${BRANCH}" == "master" ]; then echo "latest"; else echo ${BRANCH}; fi)"

docker build -f Dockerfile -t "${REPO}:${COMMIT}" .
docker tag "${REPO}:${COMMIT}" "${REPO}:${TAG}"
docker tag "${REPO}:${COMMIT}" "${REPO}:travis-${TRAVIS_BUILD_NUMBER}"
docker login -e "${DOCKER_EMAIL}" -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
docker push "${REPO}"
