#!/bin/bash

set -e

start_waitpids() {
  WAITPIDS=
}

do_waitpid() {
  $@ &
  WAITPIDS="${WAITPIDS} $!"
}

reset_waitpids() {
  for waitpid in ${WAITPIDS}; do
    wait "${waitpid}" || exit 1
  done
  WAITPIDS=
}

COVER=cover
ROOT_PKG=go.uber.org/yarpc

if [[ -d "$COVER" ]]; then
	rm -rf "$COVER"
fi
mkdir -p "$COVER"

# If a package directory has a .nocover file, don't count it when calculating
# coverage.
filter=""
for pkg in "$@"; do
	if [[ -f "$GOPATH/src/$pkg/.nocover" ]]; then
		if [[ -n "$filter" ]]; then
			filter="$filter, "
		fi
		filter="$filter\"$pkg\": true"
	fi
done

i=0
start_waitpids
for pkg in "$@"; do
	i=$((i + 1))

	extracoverpkg=""
	if [[ -f "$GOPATH/src/$pkg/.extra-coverpkg" ]]; then
		extracoverpkg=$( \
			sed -e "s|^|$pkg/|g" < "$GOPATH/src/$pkg/.extra-coverpkg" \
			| tr '\n' ',')
	fi

	coverpkg=$(go list -json "$pkg" | jq -r '
		.Deps
		| . + ["'"$pkg"'"]
		| map
			( select(startswith("'"$ROOT_PKG"'"))
			| select(contains("/vendor/") | not)
			| select({'"$filter"'}[.] | not)
			)
		| join(",")
	')
	if [[ -n "$extracoverpkg" ]]; then
		coverpkg="$extracoverpkg$coverpkg"
	fi

	args=""
	if [[ -n "$coverpkg" ]]; then
		args="-coverprofile $COVER/cover.${i}.out -coverpkg $coverpkg"
	fi

  if [ -n "${SUPPRESS_COVER_PARALLEL}" ]; then
    go test -race $args "$pkg"
  else
    do_waitpid go test -race $args "$pkg"
  fi
done
reset_waitpids

gocovmerge "$COVER"/*.out > cover.out
