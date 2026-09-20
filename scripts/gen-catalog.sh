#!/usr/bin/env bash
# Generate catalog.json by scanning this repo's GitHub Releases assets.
# Requires: gh CLI (authenticated), jq, curl.
# Env: REPO, OUT
# Release tag format: apex-<kind>-<version>; asset: <kind>-<version>-<hosttag>.tar.xz
# A sidecar asset "<asset>.sha256" (hex digest) is preferred when present.
set -euo pipefail
cd "$(dirname "$0")/.."

REPO=${REPO:-$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "Apex-Studio-Dev/apex-tools")}
OUT=${OUT:-catalog.json}
mkdir -p "$(dirname "$OUT")"

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
echo '{"schema":1,"generated":"","min_cli":"0.1.0","packages":[]}' > "$tmp/cat"

page=1
while :; do
  gh api "repos/$REPO/releases?per_page=100&page=$page" > "$tmp/rel" 2>/dev/null || break
  [ "$(jq length "$tmp/rel")" -gt 0 ] || break
  # iterate releases of this page
  for tag in $(jq -r '.[].tag_name' "$tmp/rel"); do
    kind=; ver=
    case "$tag" in
      apex-platforms-*)      kind=platforms;      ver=${tag#apex-platforms-} ;;
      apex-build-tools-*)    kind=build-tools;    ver=${tag#apex-build-tools-} ;;
      apex-platform-tools-*) kind=platform-tools; ver=${tag#apex-platform-tools-} ;;
      apex-cmake-*)          kind=cmake;          ver=${tag#apex-cmake-} ;;
      apex-ndk-*)            kind=ndk;            ver=${tag#apex-ndk-} ;;
      *) continue ;;
    esac
    [ -n "$kind" ] || continue
    case "$kind" in
      platforms)      pkgpath="platforms;android-$ver" ;;
      build-tools)    pkgpath="build-tools;$ver" ;;
      platform-tools) pkgpath="platform-tools;${ver:-1.0.0}" ;;
      cmake)          pkgpath="cmake;$ver" ;;
      ndk)            pkgpath="ndk;$ver" ;;
    esac
    gh api "repos/$REPO/releases/tags/$tag" > "$tmp/rel2"
    jq -r '.assets[] | [.name, .browser_download_url, .size] | @tsv' "$tmp/rel2" > "$tmp/assets"
    while IFS=$'\t' read -r asset url size; do
      hosttag=$(echo "$asset" | sed -n "s/^$kind-$ver-//p" | sed 's/\.tar\.xz$//')
      case "$hosttag" in linux-arm64|linux-arm|linux-x64) ;; *) continue ;; esac
      sha=$(curl -fsSL "$url.sha256" 2>/dev/null | cut -d' ' -f1 || true)
      [ -n "$sha" ] || { echo "warn: no sha256 for $asset (expected sidecar $asset.sha256)" >&2; continue; }
      jq --arg p "$pkgpath" --arg v "$ver" --argjson h "[\"$hosttag\"]" \
         --arg u "$url" --arg s "$sha" --argjson sz "$size" \
        '.packages += [{path:$p, version:$v, hosts:$h, url:$u, sha256:$s, size:$sz, license:"apex-sdk-license"}]' \
        "$tmp/cat" > "$tmp/next" && mv "$tmp/next" "$tmp/cat"
    done < "$tmp/assets"
  done
  page=$((page+1))
done

jq --arg g "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.generated = $g | .packages |= sort_by(.path)' "$tmp/cat" > "$OUT"
echo "wrote $OUT ($(jq '.packages | length' "$OUT") packages)"
