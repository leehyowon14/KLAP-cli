#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
package_dir="$repo_root/bridges/macos"
artifact_dir="$package_dir/.build/artifacts"
products=(ReminderBridge CalendarBridge CategoryBridge TranscriptBridge)

# Separate scratch directories keep the two target architectures independent.
for arch in arm64 x86_64; do
  options=(--package-path "$package_dir" --build-system native -c release
    --triple "$arch-apple-macosx12.0" --scratch-path "$package_dir/.build/release-$arch")
  swift build "${options[@]}"
  binary_dir="$(swift build "${options[@]}" --show-bin-path)"
  for product in "${products[@]}"; do
    test -x "$binary_dir/$product"
  done
  if [[ "$arch" = arm64 ]]; then
    arm_dir="$binary_dir"
  else
    intel_dir="$binary_dir"
  fi
done

mkdir -p "$artifact_dir"
for product in "${products[@]}"; do
  lipo -create "$arm_dir/$product" "$intel_dir/$product" -output "$artifact_dir/$product"
  chmod +x "$artifact_dir/$product"
  codesign --force --sign - "$artifact_dir/$product"
  lipo -verify_arch arm64 x86_64 "$artifact_dir/$product"
done
bash "$repo_root/scripts/verify-bridge-metadata.sh" "$artifact_dir"
bash "$repo_root/scripts/verify-transcript-binary.sh" "$artifact_dir/TranscriptBridge"
