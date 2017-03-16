#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

# Run protoc
#
# $1: proto file
# $2: subpackage for the output file to go into
run_protoc() {
  local dir="$(dirname "${1}")"
  local subdir="${dir}/${2}"
  rm -rf "${subdir}"
  mkdir -p "${subdir}"
  protoc --go_out=. "${1}"
  mv "${1/\.proto/.pb.go}" "${subdir}"
}

run_protoc $@
