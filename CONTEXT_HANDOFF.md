# Context Handoff

## Goal

Reduce oversized Go files by mechanically extracting cohesive chunks into same-package files first. Keep behavior unchanged, avoid new packages unless clearly needed, and verify after each pass.

Preferred file size target: around 100 lines, 300 max where practical.

## Starting Pain Points

Initial largest files called out:

- `internal/screens/mail.go`: 3221 lines
- `internal/screens/calendar.go`: 2011 lines
- `internal/app/app.go`: 1869 lines
- `cmd/telex/mail.go`: 1234 lines
- `internal/screens/tasks.go`: 1089 lines

## Strategy Used

- Same package extraction only.
- No behavior changes intended.
- Keep unexported symbols unexported.
- Run `gofmt`, `go test ./...`, and `go build ./cmd/telex` after meaningful batches.
- Remove generated local `telex` binary after build checks.

## Completed Work

### `cmd/telex/mail.go`

This file was reduced from 1234 lines to about 254 lines.

Extracted files:

- `cmd/telex/mail_labels.go`
- `cmd/telex/mail_inbox.go`
- `cmd/telex/mail_outbox.go`
- `cmd/telex/mail_mailboxes.go`
- `cmd/telex/mail_conversations.go`
- `cmd/telex/mail_drafts.go`
- `cmd/telex/mail_draft_actions.go`
- `cmd/telex/mail_draft_helpers.go`
- `cmd/telex/mail_messages.go`

All CLI mail split files are under 300 lines as of the last check.

### `internal/app/app.go`

This file was reduced from 1869 lines to about 275 lines.

Extracted or updated files:

- `internal/app/services.go`: service constructors/adapters
- `internal/app/settings.go`: settings screen state/actions/cache helpers
- `internal/app/mail_admin.go`: mail admin domain/inbox adapters
- `internal/app/contacts.go`: expanded to include contact sync/actions
- `internal/app/mail.go`: mail screen registration helpers and mail action adapters
- `internal/app/mail_convert.go`: remote mail conversion helpers
- `internal/app/drive.go`: drive sync/action adapters and opener helpers
- `internal/app/notes.go`: notes sync/action adapters
- `internal/app/tasks.go`: task sync/action adapters and board link helpers
- `internal/app/calendar.go`: calendar sync/action/invitation adapters
- `internal/app/commands.go`: command palette registration and palette context helpers

`internal/app/mail.go` is still about 347 lines, slightly above target but cohesive.
`internal/app/commands.go` is about 296 lines, under the 300-line practical target but still a dense registration file.

### `internal/screens/tasks.go`

This file was reduced from 1089 lines to about 51 lines.

Extracted files:

- `internal/screens/tasks_types.go`
- `internal/screens/tasks_keys.go`
- `internal/screens/tasks_update.go`
- `internal/screens/tasks_actions.go`
- `internal/screens/tasks_key_handlers.go`
- `internal/screens/tasks_load.go`
- `internal/screens/tasks_rows.go`
- `internal/screens/tasks_views.go`
- `internal/screens/tasks_forms.go`

All task screen split files are under 300 lines as of the last check.

### `internal/screens/calendar.go`

This file was reduced from 2011 lines to about 181 lines.

Extracted files:

- `internal/screens/calendar_types.go`: function types, modes, form data, messages, selection types
- `internal/screens/calendar_keys.go`: key map and key bindings
- `internal/screens/calendar_update.go`: root `Update`
- `internal/screens/calendar_detail.go`: detail rendering and event/invitation detail helpers
- `internal/screens/calendar_actions.go`: command/action dispatch
- `internal/screens/calendar_key_handlers.go`: key and filter input handlers
- `internal/screens/calendar_import.go`: ICS file picker/import flow
- `internal/screens/calendar_forms.go`: form lifecycle and save commands
- `internal/screens/calendar_form_inputs.go`: form input conversion, validation, and form keymap/title helpers
- `internal/screens/calendar_list.go`: calendar list rendering and calendar metadata helpers
- `internal/screens/calendar_range.go`: view mode and date range helpers
- `internal/screens/calendar_filter.go`: agenda filter parsing and matching
- `internal/screens/calendar_selection.go`: selection and index clamping helpers
- `internal/screens/calendar_load.go`: load/sync/delete/import/invitation commands

All calendar screen split files are under 300 lines as of the last check.

## Verification Status

Last successful verification commands:

```sh
gofmt -w <changed files>
go test ./...
go build ./cmd/telex
rm -f telex
```

`go test ./...` and `go build ./cmd/telex` passed after the latest extraction.

## Current Known State

The current split batch is ready to commit and includes:

- CLI mail extraction under `cmd/telex/`
- App shell/domain adapter extraction under `internal/app/`
- Tasks screen extraction under `internal/screens/tasks*.go`
- Calendar screen extraction under `internal/screens/calendar*.go`
- This handoff file

After committing this batch, the remaining next target is `internal/screens/mail.go`.

Note: `AGENTS.md` appeared modified in the working tree during the latest pass, but this agent did not edit it and it should not be included in this refactor commit unless intentionally requested.

## Recommended Next Pass

1. Move to `internal/screens/mail.go`.
2. Check whether `internal/app/mail.go` should be split further; it is about 347 lines and cohesive but above target.
3. Optionally split `internal/app/commands.go` further into module-specific command registration files if desired.
4. Consider extracting `registerScreens`, `buildHome`, and `buildNews` from `internal/app/app.go` only if pushing all app files closer to 100 lines is still a priority.

## Later Screen Split Plan

### `internal/screens/tasks.go`

Suggested files:

- `tasks.go`: struct, constructors, `Init`, `Title`
- `tasks_types.go`: function types, rows, messages, selection
- `tasks_keys.go`: key map and bindings
- `tasks_update.go`: `Update`
- `tasks_actions.go`: command palette action handling and card movement
- `tasks_keys_handlers.go`: key/filter/confirm/picker handlers
- `tasks_load.go`: load/sync commands
- `tasks_forms.go`: project/card editor templates and parsing
- `tasks_rows.go`: row building, selection, list delegate
- `tasks_views.go`: view/render functions
- `tasks_external.go`: editor command helpers

### `internal/screens/calendar.go`

Suggested files:

- `calendar.go`
- `calendar_types.go`
- `calendar_keys.go`
- `calendar_update.go`
- `calendar_actions.go`
- `calendar_keys_handlers.go`
- `calendar_forms.go`
- `calendar_import.go`
- `calendar_invitation.go`
- `calendar_load.go`
- `calendar_views.go`
- `calendar_detail_render.go`
- `calendar_list.go`

### `internal/screens/mail.go`

Suggested files:

- `mail.go`
- `mail_types.go`
- `mail_keys.go`
- `mail_update.go`
- `mail_actions.go`
- `mail_keys_handlers.go`
- `mail_drafts.go`
- `mail_attachments.go`
- `mail_links.go`
- `mail_conversation.go`
- `mail_load.go`
- `mail_views.go`
- `mail_render.go`
- `mail_external.go`

## Important Constraints

- Do not introduce new packages yet; same-package extraction is much safer.
- Do not rename exported APIs or alter behavior during extraction passes.
- Preserve Bubble Tea v2 types.
- Run verification after each batch.
- If a generated `telex` binary appears from `go build ./cmd/telex`, remove it before finishing.
