#!/bin/bash

set -euo pipefail

# $1: release branch
# $2: base branch
update_release_branch() {
  git checkout ${1}
  git merge ${2}
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

main() {
  if [ $# -lt 1 ] || [ -z ${1} ] || [ ${1:0:1} == "v" ]; then
    echo "Usage: $0 <new_version>"
    echo "  new_version: the new version to release (without 'v' prefix)"
    echo "  release_branch: the branch to release from (default: dev)"
    echo "  base_branch: the branch to release to (default: master)"
    exit 1
  fi

  local new_version="${1}"
  local release_branch="${2:-dev}"
  local base_branch="${3:-master}"

  echo "Returning to development from v${new_version} on base branch ${base_branch} to release branch ${release_branch} (press enter to continue)"
  read

  echo "Merging ${base_branch} to ${release_branch}..."
  update_release_branch "${release_branch}" "${base_branch}"

  echo "Updating CHANGELOG.md..."
  update_changelog "${new_version}"

  echo "Setting and verifying version..."
  set_and_verify_version "${new_version}"
}

main $@
