#!/usr/bin/env bash
# Split a full android-sdk-<target> splice archive (android-sdk-custom output)
# into a single-component archive for make-package.sh.
# Usage: split-sdk.sh <sdk-archive> <kind> <version>
#   kind: platforms|build-tools|platform-tools
# Output: split/<kind>-<version>.tar.xz  (top dir = component folder)
set -euo pipefail
cd "$(dirname "$0")/.."

archive=${1:?archive}
kind=${2:?kind platforms|build-tools|platform-tools}
version=${3:?version}
mkdir -p split

work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
tar -xf "$archive" -C "$work"
sdkroot=$(find "$work" -mindepth 1 -maxdepth 1 -type d | head -1)
[ -n "$sdkroot" ] || { echo "no top dir in archive" >&2; exit 5; }

case "$kind" in
  platforms)      compdir="$sdkroot/platforms/android-$version" ;;
  build-tools)    compdir="$sdkroot/build-tools/$version" ;;
  platform-tools) compdir="$sdkroot/platform-tools" ;;
esac
[ -d "$compdir" ] || { echo "component dir not found: $compdir" >&2; exit 5; }

out="split/$kind-$version.tar.xz"
tar -cJf "$out" -C "$(dirname "$compdir")" "$(basename "$compdir")"
echo "$out"
