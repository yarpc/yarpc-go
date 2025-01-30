#!/bin/bash

set -euo pipefail

check_git() {
  if ! git diff --quiet; then
    echo "Working directory is not clean. Please commit or stash your changes."
    exit 1
  fi
}

# $1: release branch
# $2: base branch
prepare_branch() {
  local release_branch="${1}"
  local base_branch="${2}"

  git fetch origin "${release_branch}"
  git fetch origin "${base_branch}"
  git checkout origin/"${base_branch}"
  git checkout -B $(whoami)/release
  git merge origin/"${release_branch}"
}

# $1: new version
set_release_to_changelog() {
  local release_date=$(date +"%Y-%m-%d")
  local new_version="${1}"

  # Replace list of changes with new version
  local replace_line="## \[Unreleased\]"
  local replace_with="## \[${new_version}\] - ${release_date}"
  sed -i '' -e "s/${replace_line}/${replace_with}/" CHANGELOG.md

  # Replace link to compare changes
  local replace_line="^\[Unreleased\]: https:\/\/github.com\/yarpc\/yarpc-go\/compare\/v(.+)...HEAD$"
  local replace_with="\[${new_version}\]: https:\/\/github.com\/yarpc\/yarpc-go\/compare\/v\1...v${new_version}"

  sed -i '' -E "s/${replace_line}/${replace_with}/" CHANGELOG.md
}

# $1: new version
set_and_verify_version() {
  sed -i '' -e "s/^const Version =.*/const Version = \"${1}\"/" version.go
  SUPPRESS_DOCKER=1 make verifyversion
}

main() {
  if [ $# -lt 1 ] || [ -z ${1} ] || [ ${1:0:1} == "v" ]; then
    echo "Usage: $0 <new_version> [release_branch] [base_branch]"
    echo "  new_version: the new version to release (without 'v' prefix)"
    echo "  release_branch: the branch to release from (default: dev)"
    echo "  base_branch: the branch to release to (default: master)"
    exit 1
  fi

  local new_version="${1}"
  local release_branch="${2:-dev}"
  local base_branch="${3:-master}"

  echo "Releasing new version: v${new_version} from ${release_branch} to ${base_branch} (press enter to continue)"
  read

  echo "Checking git status..."
  check_git

  echo "Preparing release branch..."
  prepare_branch "${release_branch}" "${base_branch}"

  echo "Updating CHANGELOG.md..."
  set_release_to_changelog "${new_version}"

  echo "Setting and verifying version..."
  set_and_verify_version "${new_version}"
}

main $@
