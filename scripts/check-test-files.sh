#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
source "${DIR}/scripts/helpers.sh"
cd "${DIR}"


NO_TEST_FILE_DIRS=""
for dir in $(not_cover_ignore_dirs); do
  if [ -z "$(find "${dir}" -name '*_test\.go')" ]; then
    if [ -z "${NO_TEST_FILE_DIRS}" ]; then
      NO_TEST_FILE_DIRS="${dir}"
    else
      NO_TEST_FILE_DIRS="${NO_TEST_FILE_DIRS} ${DIR}"
    fi
  fi
done

if [ -n "${NO_TEST_FILE_DIRS}" ]; then
  echo "error: at least one _test.go file must be in all directories with go files so that they are counted for code coverage: ${NO_TEST_FILE_DIRS}" >&2
  exit 1
fi
