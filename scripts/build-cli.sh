#!/usr/bin/env bash
# Cross-build the apex CLI. Output: dist/apex-<os>-<arch>[.exe]
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
rm -f dist/apex-*
for arch in arm64 arm; do
  GOOS=linux GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${APEX_VERSION:-dev}" -o "dist/apex-linux-$arch" ./cmd/apex
done
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${APEX_VERSION:-dev}" -o dist/apex-linux-x64 ./cmd/apex
ls -la dist/
