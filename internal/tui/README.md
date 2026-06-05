# internal/tui

Future Bubble Tea application shell.

This package should contain interactive screens and view state only. It should call `internal/app` use cases for data and actions.

Planned screens:

- Dashboard
- Courses
- Assignments
- Assignment detail
- Users/settings
- Reminder sync status

The TUI must not call `internal/klas`, `internal/account`, or `internal/reminder` directly once `internal/app` is in place.
