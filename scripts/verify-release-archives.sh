#!/usr/bin/env bash

set -euo pipefail

dist_dir="${1:-dist}"
required_entries=(
  LICENSE
  README.md
  docs/ARCHITECTURE.md
)
products=(ReminderBridge CalendarBridge CategoryBridge TranscriptBridge)
bridge_dir="bridges/macos/.build/artifacts"
verification_dir="$(mktemp -d)"
trap 'rm -rf "$verification_dir"' EXIT
expected_archives=(
  Darwin_x86_64.tar.gz
  Darwin_arm64.tar.gz
  Linux_x86_64.tar.gz
  Linux_arm64.tar.gz
  Windows_x86_64.zip
)

archive_count="$(find "$dist_dir" -maxdepth 1 -type f \( -name 'klap_*.tar.gz' -o -name 'klap_*.zip' \) | wc -l | tr -d ' ')"
if [[ "$archive_count" -ne "${#expected_archives[@]}" ]]; then
  echo "expected ${#expected_archives[@]} release archives, found ${archive_count}" >&2
  exit 1
fi

archive_for_suffix() {
  local suffix="$1"
  local matches=("${dist_dir}"/klap_*_"${suffix}")
  if [[ ! -f "${matches[0]}" || ${#matches[@]} -ne 1 ]]; then
    echo "expected exactly one archive ending in ${suffix}" >&2
    return 1
  fi
  printf '%s\n' "${matches[0]}"
}

archive_manifest() {
  local archive="$1"
  case "$archive" in
    *.tar.gz) tar -tzf "$archive" ;;
    *.zip) unzip -Z1 "$archive" ;;
    *)
      echo "unsupported archive format: ${archive}" >&2
      return 1
      ;;
  esac
}

for suffix in "${expected_archives[@]}"; do
  archive="$(archive_for_suffix "$suffix")"
  manifest="$(archive_manifest "$archive" | sed 's#^\./##')"

  for entry in "${required_entries[@]}"; do
    if ! grep -Fqx "$entry" <<<"$manifest"; then
      echo "${archive} is missing ${entry}" >&2
      exit 1
    fi
  done

  if grep -Eq '\.swift$' <<<"$manifest"; then
    echo "${archive} contains obsolete Swift scripts" >&2
    exit 1
  fi
  if [[ "$suffix" == Darwin_* ]]; then
    extracted="$verification_dir/$suffix"
    mkdir -p "$extracted"
    tar -xzf "$archive" -C "$extracted"
    for product in "${products[@]}"; do
      entry="bridges/macos/$product"
      grep -Fqx "$entry" <<<"$manifest"
      test -x "$extracted/$entry"
      # Exact bytes: no rebuild or script substitution after bridge CI.
      cmp "$bridge_dir/$product" "$extracted/$entry"
      if [[ "$(uname -s)" == Darwin ]]; then
        for arch in arm64 x86_64; do
          lipo "$extracted/$entry" -verify_arch "$arch"
        done
      fi
    done
    if [[ "$(uname -s)" == Darwin ]]; then
      bash scripts/verify-bridge-metadata.sh "$extracted/bridges/macos"
      bash scripts/verify-transcript-binary.sh "$extracted/bridges/macos/TranscriptBridge"
    fi
  elif grep -q '^bridges/macos/' <<<"$manifest"; then
    echo "${archive} unexpectedly contains macOS bridges" >&2
    exit 1
  fi

  binary=klap
  if [[ "$suffix" == Windows_* ]]; then
    binary=klap.exe
  fi
  if ! grep -Fqx "$binary" <<<"$manifest"; then
    echo "${archive} is missing ${binary}" >&2
    exit 1
  fi
done

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) native_suffix=Darwin_arm64.tar.gz ;;
  Darwin-x86_64) native_suffix=Darwin_x86_64.tar.gz ;;
  Linux-aarch64 | Linux-arm64) native_suffix=Linux_arm64.tar.gz ;;
  Linux-x86_64) native_suffix=Linux_x86_64.tar.gz ;;
  *)
    echo "archive manifests verified; native artifact smoke test skipped"
    exit 0
    ;;
esac

native_archive="$(archive_for_suffix "$native_suffix")"
smoke_dir="$verification_dir/native"
mkdir -p "$smoke_dir"
tar -xzf "$native_archive" -C "$smoke_dir"
KLAP_CACHE_DIR="$smoke_dir/cache" "$smoke_dir/klap" cache status >/dev/null

echo "release archive manifests and native artifact verified"
