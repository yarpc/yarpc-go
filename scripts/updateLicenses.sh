#!/bin/bash

set -e

python "$(dirname $0)"/updateLicense.py $(go list -json $(glide nv) | jq -r '.Dir + "/" + (.GoFiles | .[])')
