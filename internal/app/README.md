# internal/app

Headless application core for both CLI and future TUI.

This package owns use-case orchestration:

- selected user resolution
- saved session loading
- one-time relogin on session expiry
- course and assignment aggregation
- reminder sync coordination

It returns structured results. CLI and TUI decide how those results are displayed.

Do not import Bubble Tea here.
Do not print to stdout here.
