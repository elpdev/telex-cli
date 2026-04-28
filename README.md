# Telex

A terminal client for [Telex](https://telex.sh) — read, write, and manage email without leaving the command line.

## Features

- **Full-screen TUI** — Browse mail, read messages, and manage drafts with a keyboard-driven interface built on Bubble Tea v2.
- **Local mail sync** — Cache mailboxes, inbox messages, drafts, and outbox locally for offline access.
- **Read and triage** — Star, archive, trash, and mark messages read/unread from both the TUI and CLI.
- **Drafts and sending** — Create and edit drafts in Markdown, attach files, and send through your Telex server.
- **Notes workspace** — Sync, browse, create, edit, and delete Markdown notes backed by Telex Notes.
- **Article extraction** — Pull readable article text from links inside emails.
- **CLI-first** — All features are also available as subcommands for scripting and automation.
- **Themes** — Switch between Phosphor, Muted Dark, and Miami appearances.

## Installation

```sh
go install github.com/elpdev/telex-cli/cmd/telex@latest
```

Or download a prebuilt binary from the [releases page](https://github.com/elpdev/telex-cli/releases).

## Quick start

Authenticate with your Telex server:

```sh
telex account auth login
```

Sync your mailboxes and messages:

```sh
telex mail sync
```

Launch the TUI:

```sh
telex
# or explicitly:
telex tui
```

## CLI usage

### Mail

```sh
# Sync local mail cache
telex mail sync

# Sync a single mailbox
telex mail sync --mailbox hello@example.com

# List cached inbox messages
telex mail inbox list --mailbox hello@example.com

# Show a cached message
telex mail inbox show <id> --mailbox hello@example.com

# List messages from the server
telex mail messages list

# Search remote messages (sender, recipients, subject, body, attachment filenames)
telex mail search invoice --sender billing --received-from 2026-04-01 --received-to 2026-04-30

# List labels and update message labels
telex mail labels list
telex mail messages labels <id> --add <label-id>
telex mail messages labels <id> --remove <label-id>

# Show a remote message
telex mail messages show <id>

# Show message body
telex mail messages body <id>

# Show a remote conversation timeline
telex mail conversations timeline <conversation-id>

# Archive, trash, star, etc.
telex mail messages archive <id>
telex mail messages trash <id>
telex mail messages star <id>
```

### Drafts

```sh
# Create a draft
telex mail drafts create --mailbox hello@example.com --subject "Hello" --to someone@example.com --body "Message here"

# List local and synced remote drafts
telex mail drafts list --mailbox hello@example.com

# Show or edit a draft
telex mail drafts show <id> --mailbox hello@example.com
telex mail drafts edit <id> --mailbox hello@example.com --body "Updated message"

# Attach a file
telex mail drafts attach <id> <file> --mailbox hello@example.com

# Send local or synced remote draft
telex mail drafts send <id> --mailbox hello@example.com
```

### Outbox

```sh
# List queued outbound messages
telex mail outbox list --mailbox hello@example.com

# Sync delivery status with the server
telex mail outbox sync --mailbox hello@example.com
```

### Notes

```sh
# Sync Notes folders and note bodies into the local cache
telex notes sync

# Show cached Notes folders and note counts
telex notes tree

# List cached notes from the Notes root or a specific folder
telex notes list
telex notes list --folder-id <folder-id>

# Show cached note metadata and Markdown body
telex notes show <id>

# Create a remote note and cache it locally
telex notes create --title "Plan" --body "# Next steps"
telex notes create --folder-id <folder-id> --title "Plan" --file ./plan.md

# Update a remote note and refresh the local cache
telex notes edit <id> --title "Updated plan"
telex notes edit <id> --file ./plan.md

# Delete a remote note and remove it from the local cache
telex notes delete <id>
```

The full-screen TUI includes a Notes screen in the sidebar and command palette. Useful keybindings:

```text
enter          open folder or note detail
esc/backspace  go back
/              filter current folder
r              reload local cache
S              sync notes
n              create note in editor
e              edit selected note in editor
x              delete selected note after confirmation
```

TUI note create/edit uses `TELEX_NOTES_EDITOR` first, then `VISUAL`, then `EDITOR`:

```sh
export TELEX_NOTES_EDITOR=typora
```

On macOS, GUI editors should wait until the edit window closes so Telex can read the saved file before removing its temporary copy:

```sh
export TELEX_NOTES_EDITOR="open -W -n -a Typora"
```

Drive file opening uses `OPENER` when set. To open Drive Markdown files in Typora on macOS:

```sh
export OPENER="open -a Typora"
```

### Mailboxes

```sh
# Show mailbox overview
telex mail mailboxes

# Sync mailbox folders to local filesystem
telex mail mailboxes sync
```

## Development

```sh
go run ./cmd/telex
```

## Test

```sh
go test ./...
```

## Snapshot Release Build

```sh
goreleaser release --snapshot --clean
```

## Docker

```sh
docker run --rm -it ghcr.io/elpdev/telex:latest
```

## Release

```sh
git tag v0.1.0
git push origin v0.1.0
```

## Version

```sh
telex --version
```
