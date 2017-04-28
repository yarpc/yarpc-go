#!/bin/bash

set -e

if echo "${GOPATH}" | grep : >/dev/null; then
	echo "error: GOPATH must be one directory, but has multiple directories separated by colons: ${GOPATH}" >&2
	exit 1
fi

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
	if ! ls "${GOPATH}/src/${pkg}" | grep _test\.go$ >/dev/null; then
		continue
	fi
	i=$((i + 1))

	extracoverpkg=""
	if [[ -f "$GOPATH/src/$pkg/.extra-coverpkg" ]]; then
		extracoverpkg=$( \
			sed -e "s|^|$pkg/|g" < "$GOPATH/src/$pkg/.extra-coverpkg" \
			| tr '\n' ',')
	fi

	coverpkg=$(go list -json "$pkg" | jq -r '
		.Deps + .TestImports + .XTestImports
		| . + ["'"$pkg"'"]
		| unique
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
		args="-coverprofile $COVER/cover.${i}.out -covermode=atomic -coverpkg $coverpkg"
	fi

	do_waitpid go test -race $args "$pkg" 2>&1 \
		| grep -v 'warning: no packages being tested depend on'
done
reset_waitpids

# Merge cross-package coverage and then split the result into main and
# experimental coverages.
gocovmerge "$COVER"/*.out \
	| grep -v '/internal/examples/\|/internal/tests/' \
	| grep -v '/[a-z]\+test/' \  # packages in the form "footest"
	| grep -v 'mock_.*\.go' \  # mock files
	| tee >(grep -v /x/ > coverage.main.txt) \
	| (echo 'mode: atomic'; grep /x/) > coverage.x.txt
