# internal/tui

Bubble Tea application shell.

This package contains interactive screens and view state only. It calls `internal/app` use cases for data and actions, and must not call `internal/klas`, `internal/account`, or `internal/reminder` directly.

Current screens:

- Dashboard
- Due
- Assignments
- Notices
- Lectures
- Config

Planned follow-ups:

- Detail/open actions
- Search input
- User/settings editor
- Reminder sync status
