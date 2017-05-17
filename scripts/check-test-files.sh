#!/bin/bash

# This file generates the .nocover files and the ignore directories in .codecov.yml.
#
# It uses the following logic:
#
#   - Add every directory that contains a generated go file
#   - Add every directory and subdirectory in IGNORE_DIRS
#   - Remove every directory (but not subdirectory) in WHITELIST

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
source "${DIR}/scripts/source.sh"
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
  echo "error: at least one _test.go file must be in these directories: ${NO_TEST_FILE_DIRS}" >&2
  exit 1
fi
