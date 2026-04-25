# Notes Implementation Plan

## Goal

Add first-class CLI and TUI support for the `telex-web` Notes workspace instead of relying on generic Drive file workflows for markdown notes.

Drive already exposes Notes files as ordinary folders/files, but the Notes API provides a better product boundary:

- A managed `Notes` root folder that is created automatically.
- Folder validation that keeps notes inside the Notes workspace.
- Markdown note create/update using `title` and `body`, without direct-upload plumbing.
- Inline note body reads from `GET /api/v1/notes/:id`.
- A notes folder tree with note counts from `GET /api/v1/notes/tree`.

## Confirmed Backend API

Routes in `../telex-web/config/routes.rb`:

- `GET /api/v1/notes`
- `GET /api/v1/notes/tree`
- `GET /api/v1/notes/:id`
- `POST /api/v1/notes`
- `PATCH /api/v1/notes/:id`
- `DELETE /api/v1/notes/:id`

Payloads from `API::V1::NotesController`:

- Create/update request body: `{ "note": { "folder_id": <optional>, "title": "...", "body": "..." } }`
- List accepts optional `folder_id`; omitted or blank resolves to the Notes root folder.
- List supports normal pagination and sort with allowed sorts: `created_at`, `filename`, `updated_at`; default is `filename`.
- Invalid folders return `422` with `details.folder_id` containing `Folder must be within the Notes workspace`.

Serialized note fields from `API::V1::Serializers.note`:

- `id`
- `user_id`
- `folder_id`
- `title`
- `filename`
- `mime_type`
- `folder`
- `body`
- `created_at`
- `updated_at`

Serialized folder tree fields from `notes_folder_tree`:

- folder summary: `id`, `user_id`, `parent_id`, `name`, `source`, `metadata`, `created_at`, `updated_at`
- `note_count`
- `child_folder_count`
- `children`

## Current CLI/TUI State

What exists today:

- `ModuleNotes` exists in `internal/commands`, but no Notes screen is registered.
- Drive sync uses generic `/api/v1/folders` and `/api/v1/files`, so Notes folders/files can appear as generic Drive entries.
- Drive supports generic upload, rename, delete, download/open, and folder creation.

What is missing:

- No `internal/notes` service package.
- No Notes CLI commands.
- No Notes TUI screen.
- No note-specific cache model.
- No direct markdown edit flow backed by `POST/PATCH /api/v1/notes`.
- No Notes tree rendering or note counts.

## Proposed Product Shape

Keep Drive as generic file storage. Add Notes as an opinionated markdown workspace.

Notes should feel like a lightweight terminal notes app:

- Browse folders and markdown notes under the Notes workspace.
- Open a note in a readable markdown/detail view.
- Create a new note in `$EDITOR`.
- Edit an existing note in `$EDITOR`.
- Delete notes with confirmation.
- Sync/cache enough metadata and bodies for local-first browsing.
- Use the command palette module that already reserves `notes`.

## Implementation Phases

## Implementation Status

- Phase 1 is implemented in `internal/notes`.
- Phase 2 is implemented in `internal/notestore`.
- Phase 3 is implemented in `cmd/telex/notes.go`.
- Phase 4 is implemented in `internal/screens/notes.go` and registered in the app.
- Phase 5 command palette registration is implemented with Phase 4.
- `go test ./...` and `go build ./cmd/telex` pass after Phase 4.
- Next recommended work: Phase 6, documentation.

### Phase 1: Notes API Package

Status: complete.

Add `internal/notes`.

Models:

- `Note`
- `FolderSummary`
- `FolderTree`
- `NoteInput`
- `ListNotesParams`

Service methods:

- `ListNotes(ctx, params) ([]Note, *api.Pagination, error)`
- `NotesTree(ctx) (*FolderTree, error)`
- `ShowNote(ctx, id) (*Note, error)`
- `CreateNote(ctx, input) (*Note, error)`
- `UpdateNote(ctx, id, input) (*Note, error)`
- `DeleteNote(ctx, id) error`

Tests:

- Query construction for list pagination, folder, and sort.
- Endpoint and payload tests for create/update/delete.
- Decode test for note bodies and tree children.

Implemented files:

- `internal/notes/models.go`
- `internal/notes/services.go`
- `internal/notes/services_test.go`

### Phase 2: Notes Cache

Status: complete.

Add `internal/notestore` unless reuse of `drivestore` proves clearly simpler.

Recommended cache layout:

```text
<data-root>/notes/
  meta.toml
  folders/
    <folder-id>.toml
  notes/
    <note-id>/
      meta.toml
      body.md
```

Rationale:

- Avoid coupling Notes UX to Drive's path and file-name cache layout.
- Notes API gives stable note IDs and bodies directly.
- Folder tree can be cached independently from generic Drive sync.

Types:

- `NoteMeta`
- `FolderMeta`
- `CachedNote`
- `Store`

Store methods:

- `StoreTree(tree, syncedAt)`
- `StoreNote(note, syncedAt)`
- `ListNotes(folderID int64) ([]CachedNote, error)`
- `ReadNote(id int64) (*CachedNote, error)`
- `DeleteNote(id int64) error`
- `FolderTree() (*FolderTree, error)`

Sync strategy:

- Fetch `GET /api/v1/notes/tree`.
- Walk the tree and fetch `GET /api/v1/notes?folder_id=<id>` for each folder.
- Store note bodies from list responses because serializer includes `body`.

Tests:

- Store/read note body and metadata.
- Store folder tree and note counts.
- Delete removes note cache.
- Ordering is stable by title or updated time.

Implemented files:

- `internal/notestore/store.go`
- `internal/notestore/store_test.go`

### Phase 3: CLI Commands

Status: complete.

Add `cmd/telex/notes.go` and register under root.

Commands:

- `telex notes sync`
- `telex notes tree`
- `telex notes list [--folder-id <id>]`
- `telex notes show <id>`
- `telex notes create [--folder-id <id>] --title <title> [--body <body>|--file <path>]`
- `telex notes edit <id> [--title <title>] [--body <body>|--file <path>]`
- `telex notes delete <id>`

Optional editor commands:

- `telex notes compose [--folder-id <id>]`
- `telex notes edit-local <id>` or make `edit` open `$EDITOR` when no `--body`/`--file` is passed.

Recommended behavior:

- `list` should read local cache by default after sync.
- Add `--remote` later if needed; do not overbuild first pass.
- `show` should print metadata rows plus markdown body.
- `create/edit` should update remote first, then cache the returned note.
- `delete` should delete remote first, then remove local cache.

Tests:

- Command existence/help.
- `updated` note body from file input.
- Delete removes cached note after remote success.
- Sync stores tree and notes using a fake server.

Implemented files:

- `cmd/telex/notes.go`
- `cmd/telex/notes_test.go`
- `cmd/telex/root.go`

### Phase 4: TUI Notes Screen

Status: complete.

Add `internal/screens/notes.go` implementing `screens.Screen`.

Register in `internal/app/app.go`:

- `m.screens["notes"] = screens.NewNotes(notestore.New(m.dataPath), m.syncNotes).WithActions(...)`
- Add `notes` to preferred screen order after Drive or before Drive, depending desired product priority.

Screen state:

- Current folder ID and breadcrumb.
- Folder tree cache.
- Notes in selected folder.
- Selected row index.
- Mode: list/detail/edit confirmation.
- Search/filter string.
- Status/error.

Keybindings:

- `j/k` or arrow keys: move selection.
- `enter`: open folder or note detail.
- `esc/backspace`: parent folder or back to list.
- `/`: filter notes/folders in current folder.
- `r`: reload local cache.
- `S`: sync notes.
- `n`: new note in current folder.
- `e`: edit selected note in `$EDITOR`.
- `x`: delete selected note after confirmation.
- `y`: copy note ID/title/path if clipboard support exists; otherwise skip.

Views:

- List view: folders first, notes second, with note counts for folders.
- Detail view: title, folder, updated time, rendered markdown body.
- Small terminal behavior: no negative dimensions, truncate lines.

TUI action callbacks:

- `SyncNotesFunc`
- `CreateNoteFunc`
- `UpdateNoteFunc`
- `DeleteNoteFunc`

Editor flow:

- Use a temp `.md` file containing front matter-like title header or a simple template.
- Suggested template:

```markdown
Title: My note title

# Body starts here
```

- Parse first `Title:` line as title; rest is body.
- If title is blank, use `Untitled`.

Tests:

- Loads cached notes and opens detail.
- Navigates folder tree.
- New/edit invokes callbacks and refreshes cache.
- Delete requires confirmation.
- Sync refreshes list and status.
- Small terminal list/detail rendering does not panic.

Implemented files:

- `internal/screens/notes.go`
- `internal/screens/notes_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`

### Phase 5: Command Palette

Status: complete.

Use existing `ModuleNotes`.

Register commands:

- `go-notes`
- `notes-sync`
- `notes-new`
- `notes-edit`
- `notes-delete`
- `notes-search` if palette action can enter search mode cleanly.

Palette availability should depend on active screen and selected note.

### Phase 6: Documentation

Status: next.

Update `README.md` with:

- `telex notes sync`
- `telex notes list`
- `telex notes show <id>`
- `telex notes create --title "..." --body "..."`
- `telex notes edit <id>`
- TUI note keybindings.

## Open Design Decisions

1. Should Notes sync be separate from Drive sync?

Recommendation: yes. Notes has a distinct API and UX. Do not hide it inside Drive sync.

2. Should Drive hide the `Notes` workspace?

Recommendation: not in the first pass. It is acceptable for Drive to remain generic. If duplicate exposure becomes confusing, add a Drive filter later.

3. Should Notes support folders in the CLI first pass?

Recommendation: read/browse folders from `tree`, create notes into an existing `--folder-id`, but defer folder creation/rename/delete unless needed. The backend API only has note endpoints in the Notes namespace; folder management is generic Drive/folder API with Notes validation only inside note create/update.

4. Should Notes cache bodies by default?

Recommendation: yes. The list/show serializer includes body and notes are markdown text, so caching bodies is cheap and useful.

## Acceptance Criteria

- `telex notes sync` fetches the Notes tree and note bodies into local cache.
- `telex notes list` shows cached notes from the Notes root by default.
- `telex notes show <id>` displays note metadata and markdown body.
- `telex notes create/edit/delete` use the Notes API and keep local cache consistent.
- TUI has a `Notes` screen reachable from sidebar and command palette.
- TUI supports browse, detail, create, edit, delete, and sync.
- Tests cover service, cache, CLI, TUI, and app registration.
- `go test ./...` and `go build ./cmd/telex` pass.
