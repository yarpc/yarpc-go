#!/bin/bash

set -euo pipefail


parsePRIDFromCommit() {
  local commit="$1"

  local regexp="\(#([0-9]+)\)$" # (#1234)
  local subject=$(git log -1 --pretty=format:"%s" "${commit}")

  if [[ "${subject}" =~ ${regexp} ]]; then
    echo "${BASH_REMATCH[1]}"
  fi
}

parsePRReleaseNoteFromText() {
  txt="$1"
  release_note_regex="RELEASE NOTES:(.+)$"

  if [[ "${txt}" =~ ${release_note_regex} ]]; then
    echo "${BASH_REMATCH[1]}"
  fi
}

parsePRReleaseNoteFromRemote() {
  local pr_id="$1"

  local body=$(gh pr view "${pr_id}" --json "body" | jq -r '.body')

  if [[ "${body}" == "" ]]; then
    return
  fi

  parsePRReleaseNoteFromText "${body}"
}

formatReleaseNotes() {
  local release_note_regex="RELEASE NOTES: (.+)$"

  local commits="$@"
  local release_notes=""

  for commit in $@; do
    local pr_id=$(parsePRIDFromCommit "${commit}")

    if [[ "${pr_id}" == "" ]]; then
      continue
    fi

    local release_note=$(parsePRReleaseNoteFromRemote "${pr_id}")

    if [[ "${release_note}" == "" ]]; then
      continue
    fi

    release_notes="${release_notes}\n${release_note} (#${pr_id})"
  done

  echo -e "${release_notes}"
}

# $1: new version
set_version() {
  sed -i '' -e "s/^const Version =.*/const Version = \"${1}\"/" version.go
}
