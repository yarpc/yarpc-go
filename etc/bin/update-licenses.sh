#!/bin/bash

set -e

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

# We need to ignore internal/tests for licenses so that the golden test for
# thriftrw-plugin-yarpc can verify the contents of the generated code without
# running updateLicenses on it.
update-license $(find . -name '*.go' \
	| grep -v '^\./vendor' \
	| grep -v '/thriftrw-plugin-yarpc/internal/tests/' \
	| grep -v '/thriftrw-plugin-yarpc2/internal/tests/')
