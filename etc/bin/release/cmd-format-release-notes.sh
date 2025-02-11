#!/bin/bash

set -euo pipefail

source "$(dirname "${0}")/lib.sh"

formatReleaseNotes "$@"