#!/bin/bash

if [[ "$VERBOSE" == "1" ]]; then
	set -x
fi

if [[ -z "$2" ]]; then
	echo "USAGE: $0 DIR IMPORTPATH"
	echo ""
	echo "The binary at IMPORTPATH will be built and saved to DIR."
	exit 1
fi

function die() {
	echo "$1" >&2
	exit 1
}

outputDir="$1"
importPath="$2"

GOBIN="$outputDir" go install "$importPath" \
	|| die "Failed to build and install $importPath"
