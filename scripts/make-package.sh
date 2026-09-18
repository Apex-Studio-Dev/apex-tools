#!/usr/bin/env bash
# Normalize an upstream component archive into apex-tools release naming
# and inject SDK metadata (source.properties + package.xml).
# Usage: make-package.sh <kind> <version> <hosttag> <archive-in>
#   kind: platforms|build-tools|platform-tools|cmake|ndk
#   hosttag: linux-arm64|linux-arm|linux-x64
# Output: dist/<kind>-<version>-<hosttag>.tar.xz
set -euo pipefail
cd "$(dirname "$0")/.."

kind=${1:?kind}
version=${2:?version}
hosttag=${3:?hosttag}
archive=${4:?archive}
dist=dist
mkdir -p "$dist"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

tar -xf "$archive" -C "$work"
# normalize single top-level dir -> pkg/<name>
top=$(find "$work" -mindepth 1 -maxdepth 1 -type d | head -1)
[ -n "$top" ] || { echo "archive has no top dir" >&2; exit 5; }

case "$kind" in
  platforms)
    name="android-$version"
    ;;
  build-tools|platform-tools|cmake)
    name="$kind"
    ;;
  ndk)
    name="ndk"
    ;;
  *) echo "unknown kind $kind" >&2; exit 5 ;;
esac
mkdir -p "$work/pkg"
mv "$top" "$work/pkg/$name"

# metadata: source.properties + package.xml (idempotent-ish: overwrite ours)
major=${version%%.*}; rest=${version#*.}; minor=${rest%%.*}; micro=${rest#*.}
micro=${micro:-0}; micro=${micro#*.}
major=$(printf '%d' "$major" 2>/dev/null || echo 0)
minor=$(printf '%d' "$minor" 2>/dev/null || echo 0)
micro=$(printf '%d' "$micro" 2>/dev/null || echo 0)
case "$kind" in
  platforms)      pkgpath="platforms;$name" ;;
  build-tools)    pkgpath="build-tools;$version" ;;
  platform-tools) pkgpath="platform-tools;${version:-1.0.0}" ;;
  cmake)          pkgpath="cmake;$version" ;;
  ndk)            pkgpath="ndk;${major}.${minor}.${micro}" ;;
esac
cat > "$work/pkg/$name/package.xml" <<EOF
<?xml version="1.0" encoding="UTF-8" standalone="yes" ?>
<localPackage path="${pkgpath}" obsolete="false" schema-version="3" xmlns="http://schemas.android.com/repository/android/common/02">
    <revision><major>${major}</major><minor>${minor}</minor><micro>${micro}</micro></revision>
    <display-name>${kind} ${version}</display-name>
    <uses-license ref="apex-sdk-license" />
</localPackage>
EOF

case "$kind" in
  platforms)
    cat > "$work/pkg/$name/source.properties" <<EOF
Pkg.Desc=Android SDK Platform ${version#android-}
Pkg.Revision=${version#android-}
AndroidVersion.ApiLevel=${version#android-}
Pkg.License=apex-sdk-license
EOF
    ;;
  build-tools)
    cat > "$work/pkg/$name/source.properties" <<EOF
Pkg.Desc=Android SDK Build-Tools
Pkg.Revision=${version}
Pkg.License=apex-sdk-license
EOF
    ;;
esac

out="$dist/$kind-$version-$hosttag.tar.xz"
tar -cJf "$out" -C "$work/pkg" .
sha=$(sha256sum "$out" | cut -d' ' -f1)
size=$(stat -c%s "$out")
printf '%s\n' "$sha" > "$out.sha256"
echo "$out"
echo "$sha  $size"
