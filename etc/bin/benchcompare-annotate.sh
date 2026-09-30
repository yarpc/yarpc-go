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

# The benchmark step uploads the artifacts just before it finishes, so retry
# for a short while in case they are not visible yet.
downloaded=""
for attempt in 1 2 3 4 5 6; do
  if buildkite-agent artifact download "${OUT_DIR}/*" . ; then
    downloaded=1
    break
  fi
  echo "benchcompare-annotate: artifacts not found (attempt ${attempt}/6), retrying"
  sleep 10
done

if [ -z "${downloaded}" ]; then
  echo "benchcompare-annotate: could not download artifacts"
  buildkite-agent annotate --style warning --context benchmark-regression \
    "### Benchmarks did not run (informational, does not block merge)

Reason: the benchmark result could not be retrieved (no artifacts found from the benchmark step). See its job log." \
    || echo "benchcompare-annotate: annotate failed"
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
