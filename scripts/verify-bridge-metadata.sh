#!/usr/bin/env bash
set -euo pipefail

binary_dir="${1:?usage: verify-bridge-metadata.sh <binary-directory>}"
for product in ReminderBridge CalendarBridge CategoryBridge TranscriptBridge; do
  binary="$binary_dir/$product"
  test -x "$binary"
  codesign --verify --strict "$binary"
  for arch in $(lipo -archs "$binary"); do
    minimum_os="$(otool -arch "$arch" -l "$binary" | awk '$1 == "minos" { print $2 }')"
    test "$minimum_os" = "12.0"
    metadata="$(otool -arch "$arch" -P "$binary" | sed -n '/<?xml/,/<\/plist>/p')"
    identifier="$(printf '%s' "$metadata" | plutil -extract CFBundleIdentifier raw -o - -)"
    case "$product" in
      ReminderBridge) expected=reminder; keys=(NSRemindersUsageDescription NSRemindersFullAccessUsageDescription) ;;
      CalendarBridge) expected=calendar; keys=(NSCalendarsUsageDescription NSCalendarsFullAccessUsageDescription) ;;
      CategoryBridge) expected=category; keys=(NSRemindersFullAccessUsageDescription NSCalendarsFullAccessUsageDescription) ;;
      TranscriptBridge) expected=transcript; keys=(NSSpeechRecognitionUsageDescription) ;;
    esac
    test "$identifier" = "io.github.leehyowon14.klap.$expected"
    for key in "${keys[@]}"; do
      description="$(printf '%s' "$metadata" | plutil -extract "$key" raw -o - -)"
      test -n "$description"
    done
    # A bridge must not depend on libraries left in the build workspace.
    if otool -arch "$arch" -L "$binary" | tail -n +2 | grep -Eq '^[[:space:]]+(/Users/|/private/|@rpath/(EventKitCore|SpeechCore))'; then
      echo "$product ($arch) depends on a build-local library" >&2
      exit 1
    fi
  done
done
echo "bridge usage descriptions, identifiers and signatures verified"
