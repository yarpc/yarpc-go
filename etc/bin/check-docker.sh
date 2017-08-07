#!/bin/bash

docker version >/dev/null
if [ $? -ne 0 ]; then
  echo "-----------------------------------------------------------------" >&2
  echo "Most of our make commands run inside Docker but you do not appear to have" >&2
  echo "the Docker daemon running. Please start the Docker daemon and try again." >&2
  echo "See https://docs.docker.com/engine/installation/ to get started with Docker." >&2
  echo "Alternatively, set SUPPRESS_DOCKER=1 to run the command locally against your system." >&2
  echo "Docker is not running" >&2
  exit 1
fi
