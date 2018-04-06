#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

if echo "${GOPATH}" | grep : >/dev/null; then
	echo "error: GOPATH must be one directory, but has multiple directories separated by colons: ${GOPATH}" >&2
	exit 1
fi

ROOT_PKG=go.uber.org/yarpc
OUTFILE=coverage.txt

ignorePkgs=""
filterIgnorePkgs() {
  if [[ -z "${ignorePkgs}" ]]; then
    cat
  else
    grep -v "${ignorePkgs}"
  fi
}

# If a package directory has a .nocover file, don't count it when calculating
# coverage.
for pkg in "$@"; do
  if [[ -f "$GOPATH/src/$pkg/.nocover" ]]; then
    if [[ -n "$ignorePkgs" ]]; then
      ignorePkgs="$ignorePkgs\\|"
    fi
    ignorePkgs="$ignorePkgs$pkg/"
  fi
done

rm -f "$OUTFILE.tmp" "$OUTFILE"
go test -coverprofile "$OUTFILE.tmp" -covermode=count -coverpkg="$ROOT_PKG/..." "$@"
filterIgnorePkgs < "$OUTFILE.tmp" \
  | grep -v '/interna/examples/\|/internal/tests/\|/mocks/' \
  | grep -v '/[a-z]\+test/' \
  > "$OUTFILE"
rm -f "$OUTFILE.tmp"
