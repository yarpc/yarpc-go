#!/bin/bash

set -e

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

export GOBIN=$DIR/etc/bin
go install go.uber.org/tools/update-license

# We need to ignore internal/tests and internal/random_pkg for licenses so
# that the golden test for thriftrw-plugin-yarpc can verify the contents of
# the generated code without running updateLicenses on it. random_pkg sits
# alongside tests/ on purpose so it can be imported by tests/WITHSERVICES
# fixtures.
$GOBIN/update-license $(find . -name '*.go' \
	| grep -v '^\./vendor' \
	| grep -v '/thriftrw-plugin-yarpc/internal/tests/' \
	| grep -v '/thriftrw-plugin-yarpc/internal/random_pkg/' \
	| grep -v -e '.pb.go$' -e '.pb.yarpc.go$')
