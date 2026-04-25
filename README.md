# Telex

A terminal client for [Telex](https://telex.sh) — read, write, and manage email without leaving the command line.

## Features

- **Full-screen TUI** — Browse mail, read messages, and manage drafts with a keyboard-driven interface built on Bubble Tea v2.
- **Local mail sync** — Cache mailboxes, inbox messages, drafts, and outbox locally for offline access.
- **Read and triage** — Star, archive, trash, and mark messages read/unread from both the TUI and CLI.
- **Drafts and sending** — Create and edit drafts in Markdown, attach files, and send through your Telex server.
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

# Show a remote message
telex mail messages show <id>

# Show message body
telex mail messages body <id>

# Archive, trash, star, etc.
telex mail messages archive <id>
telex mail messages trash <id>
telex mail messages star <id>
```

### Drafts

```sh
# Create a draft
telex mail drafts create --mailbox hello@example.com --subject "Hello" --to someone@example.com --body "Message here"

# List drafts
telex mail drafts list --mailbox hello@example.com

# Show or edit a draft
telex mail drafts show <id> --mailbox hello@example.com
telex mail drafts edit <id> --mailbox hello@example.com --body "Updated message"

# Attach a file
telex mail drafts attach <id> <file> --mailbox hello@example.com

# Send
telex mail drafts send <id> --mailbox hello@example.com
```

### Outbox

```sh
# List queued outbound messages
telex mail outbox list --mailbox hello@example.com

# Sync delivery status with the server
telex mail outbox sync --mailbox hello@example.com
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
