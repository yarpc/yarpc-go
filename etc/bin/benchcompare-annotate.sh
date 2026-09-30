#!/bin/bash

# benchcompare-annotate.sh annotates the Buildkite build with the result of
# the benchcompare step. benchcompare.sh runs in a container without agent
# credentials, so it leaves the annotation text and style in
# .benchcompare/ and uploads them as artifacts. This script runs outside the
# container, downloads them, and calls "buildkite-agent annotate".
#
# Informational only: it never fails the build.

set -uo pipefail

OUT_DIR=".benchcompare"
mkdir -p "${OUT_DIR}"

if ! buildkite-agent artifact download "${OUT_DIR}/*" . ; then
  echo "benchcompare-annotate: could not download artifacts, nothing to annotate"
  exit 0
fi

if [ ! -s "${OUT_DIR}/annotation.md" ]; then
  buildkite-agent annotate --style warning --context benchmark-regression \
    "### Benchmarks did not run (informational, does not block merge)

Reason: the benchmark step produced no result. See its job log." \
    || echo "benchcompare-annotate: annotate failed"
  exit 0
fi

STYLE="$(head -n1 "${OUT_DIR}/style" 2>/dev/null)"
case "${STYLE}" in
  info|success|warning|error) ;;
  *) STYLE="info" ;;
esac

buildkite-agent annotate --style "${STYLE}" --context benchmark-regression \
  <"${OUT_DIR}/annotation.md" || echo "benchcompare-annotate: annotate failed"
exit 0
