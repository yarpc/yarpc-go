#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

IGNORE_DIRS="\
  internal/crossdock \
  internal/examples \
  internal/service-test \
  transport/x/cherami/example"

WHITELIST="\
  api/peer \
  api/peer/peertest \
  api/middleware/middlewaretest \
  api/transport \
  api/transport/transporttest \
  encoding/thrift \
  encoding/thrift/thriftrw-plugin-yarpc \
  internal/crossdock \
  internal/interpolate \
  internal/sync \
  transport/x/redis/redistest"

is_ignore_dir() {
  for i in ${IGNORE_DIRS}; do
    if echo "${1}" | grep "^${i}" >/dev/null; then
      return 0
    fi
  done
  return 1
}

is_whitelisted() {
  for w in ${WHITELIST}; do
    if [ "${1}" == "${w}" ]; then
      return 0
    fi
  done
  return 1
}

get_ignore_dirs() {
  while read d; do
    if is_ignore_dir "${d}"; then
      echo "${d}"
    fi
  done
}

remove_whitelisted() {
  while read d; do
    if ! is_whitelisted "${d}"; then
      echo "${d}"
    fi
  done
}

dirnames() {
  local list=""
  for filename in $@; do
    local d="$(dirname "${filename}")"
    if [ -z "${list}" ]; then
      local list="${d}"
    else
      local list="${list} ${d}"
    fi
  done
  echo "${list}" | tr ' ' '\n' | sort | uniq
}

go_files() {
  find . -name '*.go' | sed 's/^\.\///' | grep -v -e ^vendor\/ -e ^\.glide\/
}

generated_go_files() {
  find $(go_files) -exec sh -c 'head -n30 {} | grep "Code generated by\|Autogenerated by\|Automatically generated by\|@generated" >/dev/null' \; -print
}

dirnames_with_go_files() {
  dirnames $(go_files)
}

dirnames_with_generated_go_files() {
  dirnames $(generated_go_files)
}

cover_ignore_dirs_not_uniq() {
  dirnames_with_generated_go_files | remove_whitelisted
  dirnames_with_go_files | get_ignore_dirs | remove_whitelisted
}

cover_ignore_dirs() {
  cover_ignore_dirs_not_uniq | sort | uniq
}

remove_existing_nocover_files() {
  find . -name \.nocover | sed 's/^\.\///' | grep -v -e ^vendor\/ -e ^\.glide\/ | xargs rm
}

add_nocover_files() {
  for d in $@; do
    touch "${d}/.nocover"
  done
}

dirs="$(cover_ignore_dirs)"

remove_existing_nocover_files
add_nocover_files ${dirs}
