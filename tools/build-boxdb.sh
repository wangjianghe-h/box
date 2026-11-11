#!/bin/sh
set -eu
cd "$(dirname "$0")/boxdb"
GOTOOLCHAIN=local GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
  go build -trimpath -buildmode=pie -ldflags="-s -w" -o ../../box/bin/boxdb .
