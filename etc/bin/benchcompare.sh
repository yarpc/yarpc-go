#!/bin/bash

# benchcompare.sh runs benchstat over the benchmarks of packages changed by a
# PR, comparing the PR base branch against the PR head, and posts the result
# as a Buildkite annotation. This is informational only: it never fails the
# build, on a comparison error or on a detected regression alike.
#
# When the comparison does not run, for any reason, the annotation says so and
# gives the reason, so a missing result is never silent.
#
# If GITHUB_TOKEN is set (and this is a PR build), the same text is also posted
# as a comment on the pull request. One comment is kept per PR and edited in
# place on every build. GITHUB_REPO (owner/repo) defaults to yarpc/yarpc-go.
# Buildkite does not pass secrets to builds from forks, so on those the token
# is unset and only the annotation is written.

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

# post_pr_comment creates or updates the benchcompare comment on the PR. It is
# best effort: any failure is logged and ignored so it never fails the build.
post_pr_comment() {
  local body="${1}"
  local marker="<!-- benchcompare -->"

  if [ -z "${GITHUB_TOKEN:-}" ]; then
    echo "benchcompare: GITHUB_TOKEN is not set, not posting a PR comment"
    return 0
  fi
  if [ "${BUILDKITE_PULL_REQUEST:-false}" = "false" ]; then
    return 0
  fi
  if ! command -v jq >/dev/null 2>&1; then
    echo "benchcompare: jq not found, not posting a PR comment"
    return 0
  fi

  local api="https://api.github.com/repos/${GITHUB_REPO:-yarpc/yarpc-go}/issues"
  local auth="Authorization: Bearer ${GITHUB_TOKEN}"
  local payload comment_id

  payload="$(jq -n --arg b "${marker}
${body}" '{body: $b}')" || return 0

  comment_id="$(curl -fsS -H "${auth}" "${api}/${BUILDKITE_PULL_REQUEST}/comments?per_page=100" \
    | jq -r --arg m "${marker}" '[.[] | select(.body | contains($m))][0].id // empty')" || comment_id=""

  if [ -n "${comment_id}" ]; then
    curl -fsS -X PATCH -H "${auth}" -d "${payload}" "${api}/comments/${comment_id}" >/dev/null \
      || echo "benchcompare: failed to update PR comment ${comment_id}"
  else
    curl -fsS -X POST -H "${auth}" -d "${payload}" "${api}/${BUILDKITE_PULL_REQUEST}/comments" >/dev/null \
      || echo "benchcompare: failed to create PR comment"
  fi
  return 0
}

annotate() {
  local style="${1}"
  local body="${2}"
  if ! command -v buildkite-agent >/dev/null 2>&1 || \
      ! echo "${body}" | buildkite-agent annotate --style "${style}" --context benchmark-regression; then
    echo "${body}"
  fi
  post_pr_comment "${body}"
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

if [ "${BUILDKITE_PULL_REQUEST:-false}" = "false" ]; then
  not_run info "this is not a pull request build, so there is no base branch to compare against."
fi

BASE_BRANCH="${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-}"
if [ -z "${BASE_BRANCH}" ]; then
  not_run warning "\`BUILDKITE_PULL_REQUEST_BASE_BRANCH\` is not set, so the base branch is unknown."
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

if ! go test -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS} >"${BENCH_HEAD_TXT}"; then
  not_run warning "benchmarks failed to run on the PR head for:${CHANGED_PKGS}. See the job log."
fi

if ! git worktree add --detach "${BASE_WORKTREE}" "${BASE_SHA}" >/dev/null; then
  not_run warning "could not check out the merge base ${BASE_SHA} for comparison."
fi

if ! (cd "${BASE_WORKTREE}" && go test -run='^$' -bench=. -benchmem -count="${BENCH_COUNT}" -benchtime="${BENCH_TIME}" ${CHANGED_PKGS}) >"${BENCH_BASE_TXT}"; then
  not_run warning "benchmarks failed to run on \`${BASE_BRANCH}\` at ${BASE_SHA} for:${CHANGED_PKGS}. The benchmarks may be new in this PR. See the job log."
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
