#!/usr/bin/env bash

# This script will update the dependencies for the branch that it was executed
# with. If the update didn't cause any generated code to change and the tests
# continue to pass, the change will be pushed directly to the branch. If the
# generated code changed or the tests failed, the change will be pushed to a
# new branch and a pull request will be created.
#
# Variables to control the behavior of this script:
#
# GITHUB_USER (required)
#   GitHub username used to authenticate requests.
# GITHUB_TOKEN (required)
#   GitHub token to authenticate $GITHUB_USER.
# GITHUB_REPO (optional)
#   GitHub repository in the form $user/$repo. This will usually be inferred
#   automatically from BUILDKITE_REPO.
# GIT_REMOTE (optional)
#   Name of the git remote to which changes will be pushed. Defaults to
#   "origin".
#
# The following variables set by BuildKite are also accepted.
#
#   BUILDKITE_BRANCH
#   BUILDKITE_BUILD_CREATOR (optional)
#   BUILDKITE_BUILD_CREATOR_EMAIL (optional)
#   BUILDKITE_BUILD_NUMBER (optional)
#   BUILDKITE_REPO (optional)
#
# See https://buildkite.com/docs/builds/environment-variables for what they mean.

set -euo pipefail

if [ -z "${GITHUB_USER:-}" ] || [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "GITHUB_USER or GITHUB_TOKEN is unset."
  echo "Please set these variables."
  exit 1
fi

if [ -z "${BUILDKITE_BRANCH:-}" ]; then
  echo "BUILDKITE_BRANCH is unset. Is this running on BuildKite?"
  exit 1
fi

REMOTE="${GIT_REMOTE:-origin}"
BRANCH="$BUILDKITE_BRANCH"

GITHUB_REPO=${GITHUB_REPO:-}
if [ -z "$GITHUB_REPO" ]; then
  case "${BUILDKITE_REPO:-}" in
    "git@github.com:"*)
      GITHUB_REPO="${BUILDKITE_REPO#git@github.com:}"
      GITHUB_REPO="${GITHUB_REPO%.git}"
      ;;
    "https://github.com/"*)
      GITHUB_REPO="${BUILDKITE_REPO#https://github.com/}"
      GITHUB_REPO="${GITHUB_REPO%.git}"
      ;;
    *)
      echo "Could not determine GITHUB_REPO from BUILDKITE_REPO."
      echo "You can set it explicitly if you're not running this from CI."
      exit 1
  esac
fi

# Need this to be able to commit.
FALLBACK_EMAIL="$GITHUB_USER@users.noreply.github.com"
git config user.email "${BUILDKITE_BUILD_CREATOR_EMAIL:-$FALLBACK_EMAIL}"
git config user.name "${BUILDKITE_BUILD_CREATOR:-$GITHUB_USER}"

# When pushing over ssh, automatically add the host to known_hosts instead of
# prompting with,
#
#   The authenticity of host '...' can't be established.
#   RSA key fingerprint is ...
#   Are you sure you want to continue connecting (yes/no)?
git config core.sshCommand "ssh -o StrictHostKeyChecking=no"

now() {
  date +%Y-%m-%dT%H:%M:%S
}

git_status()
{
  # BuildKite's docker-compose plugin generates a fake docker-compose so we
  # need to ignore it anytime we do git status.
  git status "$@" | grep -v '?? docker-compose.buildkite'
}

if [ -n "$(git_status --porcelain)" ]; then
  echo "Working tree is dirty."
  echo "Please verify that you don't have any uncommitted changes."
  git status
  exit 1
fi

echo "Updating dependencies"
make glide-up

case "$(git_status --porcelain)" in
  "")
    echo "Nothing changed. Exiting."
    exit 0
    ;;
  " M glide.lock")
    echo "glide.lock changed"
    # Keep going
    ;;
  *)
    echo "Unexpected changes after a glide up:"
    git_status
    exit 1
esac

git add glide.lock
git commit -m "Update dependencies at $(now)"

echo "Updating generated code"
# make generate

#TODO: Uncomment the following block
# # We want to push directly to the remote only if the generated code did not
# # change and all tests pass.
# if [ -z "$(git_status --porcelain)" ]; then
#   if make lint test examples; then
#     echo "Generated code did not change and the tests passed."
#     echo "Pushing changes and exiting."
#     git push "$REMOTE" HEAD:"$BRANCH"
#     exit 0
#   fi
# else
#   # Check in the generated code ignoring the BuildKite docker-compose.
#   git add -A
#   git rm --cached docker-compose.buildkite*
#   git commit -m "Update generated code at $(now)"
# fi

PR_BRANCH=""
if [ -z "${BUILDKITE_BUILD_NUMBER:-}" ]; then
  # Use a different branch namespace if we're not running in BuildKite.
  PR_BRANCH="update-deps/local/$(now)"
else
  PR_BRANCH="update-deps/buildkite/$BUILDKITE_BUILD_NUMBER"
fi

echo "Creating a pull request using branch $PR_BRANCH"
git push "$REMOTE" HEAD:"refs/heads/$PR_BRANCH"

MESSAGE=$(echo 'I tried to update the dependencies but either the generated
code changed or some tests failed, so I need someone to validate or fix this
change.

Thanks!' | perl -p -e 's/\n/\\n/g')

curl --user "$GITHUB_USER:$GITHUB_TOKEN" -X POST \
  --data @- "https://api.github.com/repos/$GITHUB_REPO/pulls" <<EOF
{
  "title": "Update dependencies on $(now)",
  "head": "$PR_BRANCH",
  "base": "$BRANCH",
  "body": "$MESSAGE",
  "maintainer_can_modify": true
}
EOF
