#!/bin/bash

# This file generates the .nocover files and the ignore directories in .codecov.yml.
#
# It uses the following logic:
#
#   - Add every directory that contains a generated go file
#   - Add every directory and subdirectory in IGNORE_DIRS (in helpers.sh)
#   - Remove every directory (but not subdirectory) in WHITELIST (in helpers.sh)

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
source "${DIR}/etc/bin/helpers.sh"
cd "${DIR}"

remove_existing_nocover_files() {
  find . -name \.nocover | sed 's/^\.\///' | grep -v -e '^vendor/' | xargs rm
}

add_nocover_files() {
  for d in $@; do
    touch "${d}/.nocover"
  done
}

generate_codecov_file() {
  local tmpfile="$(mktemp)"
  head -$(grep -n ^ignore: .codecov.yml | cut -f 1 -d :) .codecov.yml > "${tmpfile}"
  for d in $@; do
    echo " - /${d}/" >> ${tmpfile}
  done
  mv "${tmpfile}" .codecov.yml
}

dirs="$(cover_ignore_dirs)"

remove_existing_nocover_files
add_nocover_files ${dirs}
generate_codecov_file ${dirs}
