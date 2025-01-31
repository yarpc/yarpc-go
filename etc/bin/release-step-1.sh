#!/bin/bash

set -euo pipefail

COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'
COLOR_NONE='\033[0m'

printBlue() {
  echo -e "${COLOR_BLUE}${1}${COLOR_NONE}"
}

printRed() {
  echo -e "${COLOR_RED}${1}${COLOR_NONE}"
}

check_git() {
  if ! git diff --quiet; then
    printRed "Working directory is not clean. Please commit or stash your changes."
    exit 1
  fi
}

# $1: dev branch
# $2: base branch
prepare_release_branch() {
  local dev_branch="${1}"
  local base_branch="${2}"

  git fetch origin "${dev_branch}"
  git fetch origin "${base_branch}"
  git checkout origin/"${base_branch}"
  git checkout -B $(whoami)/release
  git merge origin/"${dev_branch}"
}

# $1: new version
set_release_to_changelog() {
  local release_date=$(date +"%Y-%m-%d")
  local version="${1}"

  # Replace list of changes with new version
  local replace_line="## \[Unreleased\]"
  local replace_with="## \[${version}\] - ${release_date}"
  sed -i '' -e "s/${replace_line}/${replace_with}/" CHANGELOG.md

  # Replace link to compare changes
  local replace_line="^\[Unreleased\]: https:\/\/github.com\/yarpc\/yarpc-go\/compare\/v(.+)...HEAD$"
  local replace_with="\[${version}\]: https:\/\/github.com\/yarpc\/yarpc-go\/compare\/v\1...v${version}"

  sed -i '' -E "s/${replace_line}/${replace_with}/" CHANGELOG.md
}

# $1: new version
set_and_verify_version() {
  sed -i '' -e "s/^const Version =.*/const Version = \"${1}\"/" version.go
  SUPPRESS_DOCKER=1 make verifyversion
}

create_release_pr() {
  local base_branch="${1}"
  local version="${2}"

  git add version.go CHANGELOG.md
  git commit -m "Preparing release v${version}"

  gh pr create --base ${base_branch} --title "Preparing release v${version}" --web
}

main() {
  if [ $# -lt 1 ] || [ -z ${1} ] || [ ${1:0:1} == "v" ]; then
    echo "Usage: $0 <version> [dev_branch] [base_branch]"
    echo "  version: the new version to release (without 'v' prefix)"
    echo "  dev_branch: source branch with changes to be released (default: dev)"
    echo "  base_branch: target release branch (default: master)"
    exit 1
  fi

  local version="${1}"
  local dev_branch="${2:-dev}"
  local base_branch="${3:-master}"

  echo "Preparing release v${version} from ${dev_branch} to ${base_branch} (press enter to continue)"
  read

  printBlue "Checking git status..."
  check_git

  printBlue "Preparing release branch..."
  prepare_release_branch "${dev_branch}" "${base_branch}"

  printBlue "Updating CHANGELOG.md..."
  set_release_to_changelog "${version}"

  printBlue "Setting and verifying version..."
  set_and_verify_version "${version}"

  printBlue "Please validate git diff (press q to exit preview)..."
  git diff

  printBlue "(press enter to continue release)"
  read

  printBlue "Creating pull request to a ${dev_branch}..."
  create_release_pr "${dev_branch}" "${version}"
}

main $@
