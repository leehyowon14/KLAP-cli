# Release

KLAP uses GitHub Actions and GoReleaser for CI/CD.

## Versioning

- `v0.1.0`: first functional release.
- `v0.x`: early releases where CLI/TUI behavior may still change.
- `v1.0.0`: stable release after CLI contracts, settings schema, TUI state, and macOS bridge packaging are refactored.

## CI

`CI` runs on pushes to `main` and pull requests.

- Go tests run on Linux, macOS, and Windows.
- The macOS Swift transcript bridge is built on macOS.

## Release

Push a tag to start the release workflow:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow:

- builds cross-platform `klap` binaries
- creates checksums
- creates a draft GitHub Release
- updates the Homebrew formula

## Required Secrets

`GITHUB_TOKEN` is provided by GitHub Actions.

Set this repository secret before using Homebrew publishing:

- `HOMEBREW_TAP_TOKEN`: a token with write access to `leehyowon14/homebrew-klap`

## Homebrew Tap

Expected tap repository:

```text
leehyowon14/homebrew-klap
```

GoReleaser writes the formula to:

```text
Formula/klap.rb
```

Install command after release:

```sh
brew tap leehyowon14/klap
brew install klap
```

## macOS Bridge Packaging

The release archives include macOS Swift bridge scripts under:

```text
bridges/macos/
```

The Homebrew formula wraps `klap` with these environment variables on macOS:

- `KLAP_REMINDER_BRIDGE`
- `KLAP_CALENDAR_BRIDGE`
- `KLAP_CATEGORY_BRIDGE`
- `KLAP_TRANSCRIPT_BRIDGE`

`TranscriptBridge` is built in CI to catch Swift build failures, but the v0.1.0 Homebrew formula uses the script fallback for simpler packaging. Binary bridge packaging can be revisited before `v1.0.0`.
