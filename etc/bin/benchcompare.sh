#!/bin/bash

# benchcompare.sh runs benchstat over the benchmarks of packages changed by a
# PR, comparing the PR base branch against the PR head, and posts the result
# as a Buildkite annotation. This is informational only: it never fails the
# build, on a comparison error or on a detected regression alike.

set -uo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

BENCH_COUNT="${BENCH_COUNT:-6}"
BENCH_TIME="${BENCH_TIME:-1s}"
BASE_WORKTREE="$(mktemp -d -t benchcompare-base.XXXXXX)"
BENCH_BASE_TXT="$(mktemp -t benchcompare-base-out.XXXXXX)"
BENCH_HEAD_TXT="$(mktemp -t benchcompare-head-out.XXXXXX)"

cleanup() {
  git worktree remove --force "${BASE_WORKTREE}" >/dev/null 2>&1
  rm -f "${BENCH_BASE_TXT}" "${BENCH_HEAD_TXT}"
}
trap cleanup EXIT

annotate() {
  local style="${1}"
  local body="${2}"
  if ! command -v buildkite-agent >/dev/null 2>&1 || \
      ! echo "${body}" | buildkite-agent annotate --style "${style}" --context benchmark-regression; then
    echo "${body}"
  fi
}

skip() {
  echo "benchcompare: ${1}"
  exit 0
}

if [ "${BUILDKITE_PULL_REQUEST:-false}" = "false" ]; then
  skip "not a pull request build, skipping benchmark comparison"
fi

BASE_BRANCH="${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-}"
if [ -z "${BASE_BRANCH}" ]; then
  skip "BUILDKITE_PULL_REQUEST_BASE_BRANCH is not set, skipping benchmark comparison"
fi

# Fetch over anonymous HTTPS rather than through the "origin" remote: this
# runs inside the docker-compose sub-container, which has no SSH credentials
# or known_hosts of its own (those live on the outer Buildkite agent/pod), and
# yarpc-go is a public repo, so plain HTTPS needs no auth at all. Derive the
# HTTPS URL from "origin" so this also works from a fork's SSH-configured
# remote, falling back to the canonical repo if that fails.
FETCH_URL="$(git remote get-url origin 2>/dev/null | sed -E 's#^git@github\.com:#https://github.com/#; s#^ssh://git@github\.com/#https://github.com/#')"
FETCH_URL="${FETCH_URL:-https://github.com/yarpc/yarpc-go.git}"

if ! git fetch --depth=100 "${FETCH_URL}" "${BASE_BRANCH}"; then
  annotate warning "benchcompare: could not fetch ${BASE_BRANCH} from ${FETCH_URL}, skipping benchmark comparison"
  exit 0
fi

BASE_SHA="$(git merge-base HEAD FETCH_HEAD)"
if [ -z "${BASE_SHA}" ]; then
  annotate warning "benchcompare: could not find a merge base with ${BASE_BRANCH}, skipping benchmark comparison"
  exit 0
fi

# Packages with Go file changes relative to the merge base, mapped to
# packages that actually contain benchmarks.
CHANGED_DIRS="$(git diff --name-only "${BASE_SHA}" HEAD -- '*.go' | xargs -r -n1 dirname | sort -u)"
CHANGED_PKGS=""
for d in ${CHANGED_DIRS}; do
  if grep -rl '^func Benchmark' "${d}"/*_test.go >/dev/null 2>&1; then
    CHANGED_PKGS="${CHANGED_PKGS} ./${d}"
  fi
done

if [ -z "${CHANGED_PKGS}" ]; then
  skip "no changed packages contain benchmarks, skipping benchmark comparison"
fi

echo "benchcompare: comparing benchmarks for:${CHANGED_PKGS}"

if ! go test -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS} >"${BENCH_HEAD_TXT}"; then
  annotate warning "benchcompare: benchmarks failed to run on the PR head, skipping benchmark comparison"
  exit 0
fi

if ! git worktree add --detach "${BASE_WORKTREE}" "${BASE_SHA}" >/dev/null; then
  annotate warning "benchcompare: could not check out ${BASE_SHA} for comparison, skipping benchmark comparison"
  exit 0
fi

if ! (cd "${BASE_WORKTREE}" && go test -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS}) >"${BENCH_BASE_TXT}"; then
  annotate warning "benchcompare: benchmarks failed to run on ${BASE_BRANCH}@${BASE_SHA}, skipping benchmark comparison"
  exit 0
fi

DIFF="$(benchstat "${BENCH_BASE_TXT}" "${BENCH_HEAD_TXT}")"
STYLE="info"
if echo "${DIFF}" | grep -q '[+-][0-9].*%'; then
  STYLE="warning"
fi

annotate "${STYLE}" "### Benchmark comparison vs \`${BASE_BRANCH}\` (informational, does not block merge)

\`\`\`
${DIFF}
\`\`\`"
