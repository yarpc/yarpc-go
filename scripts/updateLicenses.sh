#!/bin/bash

set -e

# Regular expression for file paths that don't need licenses.
#
# We need to ignore internal/tests for licenses so that the golden test for
# thriftrw-plugin-yarpc can verify the contents of the generated code without
# running updateLicenses on it.
LICENSE_FILTER="/internal/tests/"

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

python scripts/updateLicense.py $(go list -json $(glide nv) | \
	jq -r '.Dir + "/" + (.GoFiles | .[]) | select(test("'"$LICENSE_FILTER"'") | not)')
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/second_service-remote # generated and not needed
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/thrift_test-remote # generated and not needed
