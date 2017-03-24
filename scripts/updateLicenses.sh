#!/bin/bash

set -e

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

# We need to ignore internal/tests for licenses so that the golden test for
# thriftrw-plugin-yarpc can verify the contents of the generated code without
# running updateLicenses on it.
update-license $(find . -name '*.go' | grep -v ^\.\/vendor | grep -v \/thriftrw-plugin-yarpc\/internal\/tests\/)
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/second_service-remote # generated and not needed
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/thrift_test-remote # generated and not needed
