#!/bin/bash

# benchcompare.sh runs benchstat over the benchmarks of packages changed by a
# PR, comparing the PR base branch against the PR head, and writes the result
# as Markdown. This is informational only: it never fails the build, on a
# comparison error or on a detected regression alike.
#
# When the comparison does not run, for any reason, the output says so and
# gives the reason, so a missing result is never silent.
#
# Inputs (environment):
#   PR_NUMBER        pull request number; "false" or unset means not a PR build
#   BASE_BRANCH      branch the PR targets
# Tunables (environment, all optional):
#   BENCH_COUNT, BENCH_TIME   passed to go test -count and -benchtime
#   BENCH_PHASE_TIMEOUT       time limit for each of the two go test runs
#                             (head, then base), default 18m. When hit, the
#                             result says so instead of going missing.
#   BENCHCOMPARE_FAIL_ON_REGRESSION
#                             1 to exit 1 on a large regression, default 0
#   BENCH_FAIL_THRESHOLD_PCT  how much worse counts as a large regression,
#                             default 50 (percent), applied to sec/op, B/op
#                             and allocs/op
#   BENCH_FAIL_ALPHA          benchstat p-value a regression must be at or
#                             below to count, default 0.01
# For Buildkite, BUILDKITE_PULL_REQUEST and BUILDKITE_PULL_REQUEST_BASE_BRANCH
# are used when the above are unset.
#
# Outputs, written to BENCHCOMPARE_OUT_DIR (default .benchcompare):
#   annotation.md    the Markdown result (also printed to stdout)
#   style            info, success, warning or error
#
# In CI, .github/workflows/benchcompare.yml runs this on pull_request without
# secrets and uploads the output directory as an artifact.
# .github/workflows/benchcompare-comment.yml then posts annotation.md as a PR
# comment from a trusted context, so fork PRs get the comment too.

set -uo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

BENCHCOMPARE_OUT_DIR="${BENCHCOMPARE_OUT_DIR:-.benchcompare}"
BENCH_COUNT="${BENCH_COUNT:-6}"
BENCH_TIME="${BENCH_TIME:-1s}"
BENCH_PHASE_TIMEOUT="${BENCH_PHASE_TIMEOUT:-18m}"
BENCHCOMPARE_FAIL_ON_REGRESSION="${BENCHCOMPARE_FAIL_ON_REGRESSION:-0}"
BENCH_FAIL_THRESHOLD_PCT="${BENCH_FAIL_THRESHOLD_PCT:-50}"
BENCH_FAIL_ALPHA="${BENCH_FAIL_ALPHA:-0.01}"
BASE_WORKTREE="$(mktemp -d -t benchcompare-base.XXXXXX)"
BENCH_BASE_TXT="$(mktemp -t benchcompare-base-out.XXXXXX)"
BENCH_HEAD_TXT="$(mktemp -t benchcompare-head-out.XXXXXX)"

cleanup() {
  git worktree remove --force "${BASE_WORKTREE}" >/dev/null 2>&1
  rm -f "${BENCH_BASE_TXT}" "${BENCH_HEAD_TXT}"
}
trap cleanup EXIT

# annotate records the result in BENCHCOMPARE_OUT_DIR and prints it to the log.
annotate() {
  local style="${1}"
  local body="${2}"
  mkdir -p "${BENCHCOMPARE_OUT_DIR}" 2>/dev/null \
    && printf '%s\n' "${style}" >"${BENCHCOMPARE_OUT_DIR}/style" \
    && printf '%s\n' "${body}" >"${BENCHCOMPARE_OUT_DIR}/annotation.md" \
    || echo "benchcompare: could not write ${BENCHCOMPARE_OUT_DIR}, the result will not be posted"
  echo "${body}"
}

# not_run annotates that benchmarks did not run, with the reason, then exits
# successfully. The first argument is the annotation style: "info" for an
# expected skip, "warning" for a failure to run the comparison.
not_run() {
  local style="${1}"
  local reason="${2}"
  annotate "${style}" "### Benchmarks did not run (informational, does not block merge)

Reason: ${reason}"
  exit 0
}

# run_bench runs the benchmarks of CHANGED_PKGS from the directory given as
# the first argument, bounded by BENCH_PHASE_TIMEOUT. It returns 124 on a
# timeout. Where a timeout(1) binary is available it bounds the whole run;
# otherwise go test's own per-package -timeout is the only bound.
run_bench() {
  local dir="${1}"
  local timeout_cmd
  timeout_cmd="$(command -v timeout || command -v gtimeout || true)"
  if [ -n "${timeout_cmd}" ]; then
    (cd "${dir}" && "${timeout_cmd}" "${BENCH_PHASE_TIMEOUT}" \
      go test -timeout=0 -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS})
  else
    (cd "${dir}" && \
      go test -timeout="${BENCH_PHASE_TIMEOUT}" -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS})
  fi
}

# bench_failed annotates why a benchmark run failed, telling a timeout apart
# from other failures. The first argument is the run's exit status, the second
# the output file, the third a description of what ran.
bench_failed() {
  local status="${1}"
  local out="${2}"
  local what="${3}"
  if [ "${status}" -eq 124 ] || grep -q 'panic: test timed out' "${out}" 2>/dev/null; then
    not_run warning "benchmarks timed out after ${BENCH_PHASE_TIMEOUT} ${what} for:${CHANGED_PKGS}. The PR may touch packages with many or slow benchmarks. Raise \`BENCH_PHASE_TIMEOUT\` or lower \`BENCH_COUNT\` if this keeps happening."
  fi
  not_run warning "benchmarks failed to run ${what} for:${CHANGED_PKGS}. See the job log."
}

# find_regressions reads benchstat output on stdin and prints the rows that got
# worse by at least BENCH_FAIL_THRESHOLD_PCT percent with a p-value at or
# below BENCH_FAIL_ALPHA, prefixed with their unit. Rows without a numeric
# delta ("~" for no significant change, "?" for a zero baseline or a changed
# benchmark set) and the geomean summary are never counted.
find_regressions() {
  awk -v thr="${BENCH_FAIL_THRESHOLD_PCT}" -v alpha="${BENCH_FAIL_ALPHA}" '
    /vs base/ { unit = $2; next }
    $1 == "geomean" { next }
    {
      for (i = 2; i < NF; i++) {
        if ($i ~ /^\+[0-9]+(\.[0-9]+)?%$/ && $(i + 1) ~ /^\(p=[0-9.]+$/) {
          pct = substr($i, 2, length($i) - 2) + 0
          p = substr($(i + 1), 4) + 0
          if (pct >= thr && p <= alpha) {
            printf "%s: %s %s %s)\n", unit, $1, $i, $(i + 1)
          }
          break
        }
      }
    }'
}

PR_NUMBER="${PR_NUMBER:-${BUILDKITE_PULL_REQUEST:-false}}"
if [ "${PR_NUMBER}" = "false" ]; then
  not_run info "this is not a pull request build, so there is no base branch to compare against."
fi

BASE_BRANCH="${BASE_BRANCH:-${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-}}"
if [ -z "${BASE_BRANCH}" ]; then
  not_run warning "the base branch (\`BASE_BRANCH\`) is not set, so it is unknown."
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
  not_run warning "could not fetch \`${BASE_BRANCH}\` from ${FETCH_URL}."
fi

BASE_SHA="$(git merge-base HEAD FETCH_HEAD)"
if [ -z "${BASE_SHA}" ]; then
  not_run warning "could not find a merge base with \`${BASE_BRANCH}\` in the fetched history."
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
  not_run info "no package changed by this PR contains benchmarks (\`func Benchmark...\` in a \`_test.go\` file), so there is nothing to compare."
fi

echo "benchcompare: comparing benchmarks for:${CHANGED_PKGS}"

run_bench "${DIR}" >"${BENCH_HEAD_TXT}"
status=$?
if [ "${status}" -ne 0 ]; then
  bench_failed "${status}" "${BENCH_HEAD_TXT}" "on the PR head"
fi

if ! git worktree add --detach "${BASE_WORKTREE}" "${BASE_SHA}" >/dev/null; then
  not_run warning "could not check out the merge base ${BASE_SHA} for comparison."
fi

run_bench "${BASE_WORKTREE}" >"${BENCH_BASE_TXT}"
status=$?
if [ "${status}" -ne 0 ]; then
  bench_failed "${status}" "${BENCH_BASE_TXT}" "on \`${BASE_BRANCH}\` at ${BASE_SHA} (they may be new in this PR)"
fi

DIFF="$(benchstat "base=${BENCH_BASE_TXT}" "head=${BENCH_HEAD_TXT}")"
REGRESSIONS="$(printf '%s\n' "${DIFF}" | find_regressions)"

STYLE="info"
if echo "${DIFF}" | grep -q '[+-][0-9].*%'; then
  STYLE="warning"
fi

FAILING=0
if [ "${BENCHCOMPARE_FAIL_ON_REGRESSION}" = "1" ] && [ -n "${REGRESSIONS}" ]; then
  FAILING=1
  STYLE="error"
  annotate "${STYLE}" "### Benchmark regression vs \`${BASE_BRANCH}\`: failing

These benchmarks got at least ${BENCH_FAIL_THRESHOLD_PCT}% worse (p <= ${BENCH_FAIL_ALPHA}):

\`\`\`
${REGRESSIONS}
\`\`\`

Full comparison:

\`\`\`
${DIFF}
\`\`\`"
else
  annotate "${STYLE}" "### Benchmark comparison vs \`${BASE_BRANCH}\` (informational, does not block merge)

\`\`\`
${DIFF}
\`\`\`"
fi

if [ "${FAILING}" -eq 1 ]; then
  exit 1
fi
