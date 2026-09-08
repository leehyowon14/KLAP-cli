#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
binary="${1:?usage: verify-transcript-binary.sh <TranscriptBridge>}"
fixture_dir="$repo_root/bridges/macos/testdata"
test -x "$binary"

# Empty jobs return before any Speech asset, microphone or user-file access.
plain_response="$("$binary" < "$fixture_dir/empty-request.json")"
expected_plain="$(<"$fixture_dir/empty-response.json")"
test "$plain_response" = "$expected_plain"
progress_response="$("$binary" < "$fixture_dir/empty-progress-request.json")"
expected_progress="$(<"$fixture_dir/empty-progress-response.ndjson")"
test "$progress_response" = "$expected_progress"
echo "compiled transcript JSON and NDJSON protocol verified"
