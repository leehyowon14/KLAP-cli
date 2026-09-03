# Release

KLAP uses GitHub Actions and GoReleaser for CI/CD.

The canonical repository and Go module are both:

```text
github.com/leehyowon14/KLAP-cli
```

## Versioning

- `v0.1.0`: first functional release.
- `v0.x`: early releases where CLI/TUI behavior may still change.
- `v1.0.0`: stable release after CLI contracts, settings schema, TUI state, and macOS bridge packaging are refactored.

## CI

`CI` runs on pushes to `main` and pull requests.

- Go tests run on Linux, macOS, and Windows.
- All four distributed macOS Swift scripts are typechecked and the transcript bridge is built on macOS.
- GoReleaser configuration, snapshot archive manifests, and a native release artifact are verified on Linux.

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
- updates the Homebrew Cask

## Required Secrets

`GITHUB_TOKEN` is provided by GitHub Actions.

Before the first tagged release:

1. Create the public `leehyowon14/homebrew-klap` Tap repository.
2. Set `HOMEBREW_TAP_TOKEN` to a token with write access to that repository.

The release workflow is expected to fail at the Homebrew publish step until both prerequisites exist.

## Homebrew Tap

Expected tap repository:

```text
leehyowon14/homebrew-klap
```

GoReleaser writes the Cask to:

```text
Casks/klap.rb
```

Install command after the Tap exists and the first release succeeds:

```sh
brew tap leehyowon14/klap
brew install --cask klap
```

## macOS Bridge Packaging

The release archives include macOS Swift bridge scripts under:

```text
bridges/macos/
```

The Homebrew Cask links `klap` from its staged archive. KLAP resolves that link to find the bundled scripts automatically. The following environment variables remain available as explicit overrides:

- `KLAP_REMINDER_BRIDGE`
- `KLAP_CALENDAR_BRIDGE`
- `KLAP_CATEGORY_BRIDGE`
- `KLAP_TRANSCRIPT_BRIDGE`

`TranscriptBridge` is built in CI to catch Swift build failures, while release archives currently use the typechecked `transcribe.swift` entrypoint. Shipping the same compiled bridge artifact is tracked as a later adapter cleanup.
