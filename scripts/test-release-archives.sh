#!/usr/bin/env bash
set -euo pipefail

# Negative tests operate only on temporary copies of an existing snapshot.
source_dist="${1:-dist}"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT
mkdir -p "$test_root/dist" "$test_root/extracted"
for archive in "$source_dist"/klap_*.tar.gz "$source_dist"/klap_*.zip; do
  cp "$archive" "$test_root/dist/"
done
expect_failure() {
  if bash scripts/verify-release-archives.sh "$test_root/dist" >"$test_root/output" 2>&1; then
    echo "archive verifier accepted $1" >&2
    exit 1
  fi
}

archives=("$test_root/dist"/*Darwin_arm64.tar.gz)
archive="${archives[0]}"
mv "$archive" "$test_root/original.tar.gz"
expect_failure "a missing platform archive"
tar -xzf "$test_root/original.tar.gz" -C "$test_root/extracted"
tar -czf "$archive" --exclude=bridges/macos/TranscriptBridge -C "$test_root/extracted" .
expect_failure "a missing bridge"
cp "$test_root/extracted/bridges/macos/CalendarBridge" "$test_root/extracted/bridges/macos/TranscriptBridge"
tar -czf "$archive" -C "$test_root/extracted" .
expect_failure "a substituted bridge"
cp "$test_root/original.tar.gz" "$archive"
bash scripts/verify-release-archives.sh "$test_root/dist"
echo "archive rejection tests verified"
