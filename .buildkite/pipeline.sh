#!/usr/bin/env bash

# This script allows requesting a different BuildKite pipeline by setting the
# YARPC_PIPELINE environment variable.
#
# There must be a pipeline-$YARPC_PIPELINE.yml in this directory for this to
# work.

set -euo pipefail

PIPELINE=${YARPC_PIPELINE:-default}
cat "$(dirname "$0")/pipeline-$PIPELINE.yml"
