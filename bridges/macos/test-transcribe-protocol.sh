#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
module_cache="$(mktemp -d "${TMPDIR:-/tmp}/klap-swift-cache.XXXXXX")"
trap 'rm -rf "$module_cache"' EXIT
export CLANG_MODULE_CACHE_PATH="$module_cache/clang"
export SWIFT_MODULECACHE_PATH="$module_cache/swift"

plain_response="$(swift "$script_dir/transcribe.swift" < "$script_dir/testdata/empty-request.json")"
expected_plain="$(<"$script_dir/testdata/empty-response.json")"
if [[ "$plain_response" != "$expected_plain" ]]; then
  echo "unexpected transcript response: $plain_response" >&2
  exit 1
fi

progress_response="$(swift "$script_dir/transcribe.swift" < "$script_dir/testdata/empty-progress-request.json")"
expected_progress="$(<"$script_dir/testdata/empty-progress-response.ndjson")"
if [[ "$progress_response" != "$expected_progress" ]]; then
  echo "unexpected transcript progress response: $progress_response" >&2
  exit 1
fi
