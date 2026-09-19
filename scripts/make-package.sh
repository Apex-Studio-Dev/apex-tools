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

# Google-style display-name; platforms use the Android OS version (37.x -> 17).
case "$kind" in
  platforms)
    pv=${version#android-}; dver=${pv%%.*}
    case "$dver" in
      30) androidver=11 ;; 31) androidver=12 ;; 32) androidver=12 ;;
      33) androidver=13 ;; 34) androidver=14 ;; 35) androidver=15 ;;
      36) androidver=16 ;; 37) androidver=17 ;;
      *) androidver=$(printf '%d' "$dver" 2>/dev/null || echo "$dver") ;;
    esac
    dn="Android SDK Platform $androidver"
    ;;
  build-tools)    dn="Android SDK Build-Tools ${version%%.*}" ;;
  platform-tools) dn="Android SDK Platform-Tools" ;;
  cmake)          dn="CMake $version" ;;
  ndk)            dn="NDK (Side by side) $version" ;;
esac

# package.xml mirrors Google's repository.xml: namespaced root, embedded
# license text, genericDetailsType and a uses-license reference.
license_file=licenses/apex-sdk-license
[ -f "$license_file" ] || { echo "missing $license_file" >&2; exit 5; }
{
  printf '%s\n' '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
  printf '<ns2:repository xmlns:ns2="http://schemas.android.com/repository/android/common/02" xmlns:ns3="http://schemas.android.com/repository/android/common/01" xmlns:ns4="http://schemas.android.com/repository/android/generic/01" xmlns:ns5="http://schemas.android.com/repository/android/generic/02" xmlns:ns6="http://schemas.android.com/sdk/android/repo/addon2/01" xmlns:ns7="http://schemas.android.com/sdk/android/repo/addon2/02" xmlns:ns8="http://schemas.android.com/sdk/android/repo/addon2/03" xmlns:ns9="http://schemas.android.com/sdk/android/repo/repository2/01" xmlns:ns10="http://schemas.android.com/sdk/android/repo/repository2/02" xmlns:ns11="http://schemas.android.com/sdk/android/repo/repository2/03" xmlns:ns12="http://schemas.android.com/sdk/android/repo/sys-img2/04" xmlns:ns13="http://schemas.android.com/sdk/android/repo/sys-img2/03" xmlns:ns14="http://schemas.android.com/sdk/android/repo/sys-img2/02" xmlns:ns15="http://schemas.android.com/sdk/android/repo/sys-img2/01">'
  printf '<license id="apex-sdk-license" type="text">'
  sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' "$license_file"
  printf '%s\n' '</license>'
  printf '<localPackage path="%s" obsolete="false">\n' "$pkgpath"
  printf '    <type-details xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="ns5:genericDetailsType"/>\n'
  printf '    <revision><major>%s</major><minor>%s</minor><micro>%s</micro></revision>\n' "$major" "$minor" "$micro"
  printf '    <display-name>%s</display-name>\n' "$dn"
  printf '    <uses-license ref="apex-sdk-license"/>\n'
  printf '%s\n' '</localPackage>'
  printf '%s\n' '</ns2:repository>'
} > "$work/pkg/$name/package.xml"

case "$kind" in
  platforms|build-tools)
    sp="$work/pkg/$name/source.properties"
    if [ ! -f "$sp" ]; then
      if [ "$kind" = platforms ]; then
        cat > "$sp" <<EOF
Pkg.Desc=Android SDK Platform ${version#android-}
Pkg.Revision=${version#android-}
AndroidVersion.ApiLevel=${version#android-}
EOF
      else
        cat > "$sp" <<EOF
Pkg.Desc=Android SDK Build-Tools
Pkg.Revision=${version}
EOF
      fi
    elif [ "$kind" = platforms ]; then
      sed -i "s|\${PLATFORM_VERSION}|$androidver|g" "$sp"
    fi
    ;;
esac

out="$dist/$kind-$version-$hosttag.tar.xz"
tar -cJf "$out" -C "$work/pkg" .
sha=$(sha256sum "$out" | cut -d' ' -f1)
size=$(stat -c%s "$out")
printf '%s\n' "$sha" > "$out.sha256"
echo "$out"
echo "$sha  $size"
