#!/bin/bash

# This installs all hooks inside ./hooks into .git/hooks, skipping anything
# that's already there.

set -e

ROOT=$(pwd)

if [[ ! -d "$ROOT/.git" ]]; then
	echo "Please run this from the project root."
	exit 1
fi

find "$ROOT/hooks" -type file | while read hook; do
	name=$(basename "$hook")
	dest="$ROOT/.git/hooks/$name"
	if [[ -f "$dest" ]]; then
		echo "Skipping hook $name because it's already installed."
	else
		ln -s "$hook" "$dest"
		echo "Installed hook $name."
	fi
done
