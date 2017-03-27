#!/bin/bash

set -e

PROTOBUF_VERSION=3.2.0
PROTOBUF_BASENAME="protobuf-cpp-${PROTOBUF_VERSION}"

cd /home/travis
wget "https://github.com/google/protobuf/releases/download/v${PROTOBUF_VERSION}/${PROTOBUF_BASENAME}.tar.gz"
tar xzf "${PROTOBUF_BASENAME}.tar.gz"
cd "protobuf-${PROTOBUF_VERSION}"
./configure --prefix=/home/travis && make -j2 && make install
