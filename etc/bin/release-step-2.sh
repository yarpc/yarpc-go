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

# $1: base branch
pull_base_branch() {
  local base_branch="${1}"

  git checkout "${base_branch}"
  git pull origin "${base_branch}"
}

# $1: base branch
# $2: new version
create_release() {
  local base_branch="${1}"
  local version="${2}"

  gh release create "v${version}" --latest --target "${base_branch}" --title "v${version}"
}

# $1: dev branch
# $2: base branch
prepare_back_to_dev_branch() {
  local dev_branch="${1}"
  local base_branch="${2}"

  git checkout "${dev_branch}"
  git checkout -B $(whoami)/back-to-development
  git merge origin/"${base_branch}"
}

# $1: new version
update_changelog() {
  local replace_line="(## \[${1}\].+)$"
  local replace_with="## \[Unreleased\]\n- No changes yet.\n\n\1"
  sed -i '' -E "s/${replace_line}/${replace_with}/" CHANGELOG.md

  local replace_line="^(\[${1}\]: https:.+)$"
  local replace_with="\[Unreleased\]: https:\/\/github.com\/yarpc\/yarpc-go\/compare\/v${1}...HEAD\n\1"
  sed -i '' -E "s/${replace_line}/${replace_with}/" CHANGELOG.md
}

# $1: new version
set_and_verify_version() {
  local arr=($(echo "${1}" | tr '.' '\n'))
  local next_ver="${arr[0]}.$((arr[1]+1)).0-dev"

  sed -i '' -e "s/^const Version =.*/const Version = \"${next_ver}\"/" version.go
  SUPPRESS_DOCKER=1 make verifyversion
}

# $1: release branch
create_return_to_dev_pr() {
  local dev_branch="${1}"

  git add version.go CHANGELOG.md
  git commit -m "Back to development"

  gh pr create --base ${dev_branch} --title "Back to development" --web
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

  echo "Releasing new version v${version} in base branch ${base_branch} with all changes from dev branch ${dev_branch} (press enter to continue)"
  read

  printBlue "Pulling base branch..."
  pull_base_branch "${base_branch}"

  printBlue "Creating release..."
  create_release "${base_branch}" "${version}"

  printBlue "Returning to development state branch ${dev_branch}... (press enter to continue)"
  read

  prepare_back_to_dev_branch "${dev_branch}" "${base_branch}"

  printBlue "Updating CHANGELOG.md..."
  update_changelog "${version}"

  printBlue "Setting and verifying version..."
  set_and_verify_version "${version}"

  printBlue "Creating pull request to a ${dev_branch}..."
  create_return_to_dev_pr "${dev_branch}"
}

main $@
