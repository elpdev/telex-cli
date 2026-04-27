# Bubbles v2 Component Migration

## Goal

Move Telex onto Bubbles v2-native components where they fit, while keeping domain-specific screens stable and testable.

Use v2 import paths for Telex-owned UI code:

```go
charm.land/bubbles/v2/...
```

Keep any v1 compatibility isolated at external package boundaries only.

## Current Status

- Phase 1 is complete and committed as `89b7170 use bubbles v2 key bindings`.
- Phase 2 is complete and committed as `dc30438 use bubbles help component`.
- Phase 3 is implemented locally and verified, but not committed yet.
- Phase 4 is implemented locally and verified, but not committed yet.
- Phase 5 is implemented locally and verified, but not committed yet.
- `go test ./...` passes after Phase 5.
- `go build ./cmd/telex` passes after Phase 5.
- The generated `./telex` binary from build verification was removed.
- `PLAN_HANDOFF.md` is being replaced with this migration tracker.

## Phase 1: Bubbles v2 Key Bindings

Status: complete and committed.

Commit:

```text
89b7170 use bubbles v2 key bindings
```

What changed:

- Telex-owned key bindings now use `charm.land/bubbles/v2/key`.
- `internal/screens.Screen` is now a Telex-owned v2-compatible interface instead of an alias to `github.com/elpdev/tuimod.Screen`.
- Hacker News screens are wrapped at the app boundary because `github.com/elpdev/hackernews` and `github.com/elpdev/tuimod` still expose v1 key bindings.
- Legacy v1 key usage is isolated to `internal/app/hackernews.go` as `legacykey` for binding conversion.
- Calendar invitation detail behavior was fixed after tests exposed that loaded invitation details could be hidden when the refreshed agenda list was empty.

Verification completed:

```sh
go test ./...
go build ./cmd/telex
```

## Phase 2: Bubbles v2 Help Component

Status: complete and committed.

Commit:

```text
dc30438 use bubbles help component
```

Files changed:

- `internal/app/app.go`
- `internal/app/view.go`
- `internal/components/footer/footer.go`

What changed:

- Added a root `help.Model` to `app.Model`.
- Footer rendering now uses `help.Model.ShortHelpView`.
- Help overlay key rendering now uses `help.Model.FullHelpView`.
- Existing Telex modal, footer, and theme shell styling is preserved.

Verification completed:

```sh
go test ./...
go build ./cmd/telex
```

## Phase 3: Bubbles v2 Viewport For Simple Scrolling

Status: implemented locally, verified, not committed.

Target component:

```go
charm.land/bubbles/v2/viewport
```

Recommended first target:

- `internal/screens/logs.go`

Why start there:

- It is a simple read-only scrollable text screen.
- It currently uses manual `offset` state and line slicing.
- It is a low-risk pattern proof before touching Mail.

Expected work:

- Added `viewport.Model` to `Logs`.
- Replaced manual offset scrolling with viewport scroll methods.
- Set viewport width, height, and content during rendering.
- Preserved `up/k` and `down/j` behavior.

Files changed:

- `internal/screens/logs.go`

Verification completed:

```sh
go test ./...
go build ./cmd/telex
```

Next action:

- Commit Phase 3, likely with:

```text
use viewport for logs screen
```

## Phase 4: Bubbles v2 Viewport For Mail Readers

Status: implemented locally, verified, not committed.

Targets:

- Mail detail body currently using `detailScroll`.
- Mail article reader currently using `articleScroll`.
- Mail conversation body currently using `conversationScroll`.

Why after Logs:

- Mail has multiple modes and more behavior around selected messages, links, attachments, article extraction, and conversations.
- Proving the viewport integration in Logs first reduces risk.

What changed:

- Replaced Mail detail, article, and conversation body scroll integers with `viewport.Model` state.
- Kept mode transitions resetting scroll for detail open/back, article extraction, conversation load/back, and conversation entry navigation.
- Preserved existing `up/k` and `down/j` behavior.
- Kept headers, separators, and footer hints outside the viewport so Mail reader layout remains stable.
- Removed obsolete manual max-scroll helpers.

Files changed:

- `internal/screens/mail.go`

Verification completed:

```sh
go test ./...
go build ./cmd/telex
```

Next action:

- Commit Phase 3 and Phase 4 when ready, likely as separate commits.

## Phase 5: Evaluate Bubbles v2 Filepicker

Status: implemented locally, verified, not committed.

Target component:

```go
charm.land/bubbles/v2/filepicker
```

Current custom picker:

```text
internal/components/filepicker/filepicker.go
```

Current users:

- Calendar ICS import
- Drive upload
- Mail attachment save/open flows

Required parity to check:

- Open-file mode.
- Open-directory mode.
- Start in current working directory.
- Cancel with `esc`.
- Select files for upload/import.
- Select directories where needed.
- Show or toggle hidden files.
- Restrict Calendar import to `.ics` if practical.
- Preserve root restriction if still required.

What changed:

- Replaced the custom file listing/filtering implementation in `internal/components/filepicker` with Bubbles v2 `filepicker.Model`.
- Updated Calendar ICS import, Drive upload, and Mail draft attachment flows to initialize the async Bubbles filepicker command and route picker messages while active.
- Kept a minimal local action type so existing screens can close picker flows on select/cancel.
- Preserved `esc` as cancel at the Telex picker boundary to avoid trapping users in picker mode.
- Configured the Bubbles filepicker to show hidden files by default.
- Dropped custom picker parity for inline filtering, root-restricted navigation, dot hidden-file toggling, and home-key navigation.

Files changed:

- `internal/components/filepicker/filepicker.go`
- `internal/components/filepicker/filepicker_test.go`
- `internal/screens/calendar.go`
- `internal/screens/calendar_test.go`
- `internal/screens/drive.go`
- `internal/screens/drive_test.go`
- `internal/screens/mail.go`
- `internal/screens/mail_test.go`

Verification completed:

```sh
go test ./...
go build ./cmd/telex
```

## Phase 6: Selective Bubbles v2 List Adoption

Status: not started.

Target component:

```go
charm.land/bubbles/v2/list
```

Important guidance:

- Do not blindly migrate every list-like screen.
- Use `list` where it simplifies filtering, pagination, selection, and help.
- Keep custom rendering where domain-specific behavior is clearer.

Suggested candidate order:

- Settings theme selector
- Calendar calendars view
- Notes list
- Drive file list
- MailAdmin domain/inbox lists
- Mail messages last, if ever

Do not replace `github.com/elpdev/tuipalette` with generic `list`; the command palette is already purpose-built.

## Phase 7: Dependency Cleanup

Status: not started.

Do after component migrations settle.

Checklist:

- Search for remaining `github.com/charmbracelet/bubbles/...` imports.
- Keep legacy v1 imports only where external packages still require them.
- Run `go mod tidy` only when the v1 dependency is truly removable.
- Run `go test ./...`.
- Run `go build ./cmd/telex`.
- Remove generated `./telex` binary after build verification.

## Verification Commands

Run before committing each phase:

```sh
go test ./...
go build ./cmd/telex
rm -f telex
```

Check worktree state:

```sh
git status --short
```

## Notes

- `github.com/elpdev/tuimod` currently has no v2-compatible release; it still exposes `github.com/charmbracelet/bubbles/key` in `Screen.KeyBindings()`.
- The current workaround is a local Telex `screens.Screen` interface plus a Hacker News wrapper in `internal/app/hackernews.go`.
- If `tuimod` and `hackernews` gain v2-native screen contracts later, remove the wrapper and legacy key conversion.
