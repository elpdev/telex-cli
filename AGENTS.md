# AGENTS.md

Guidance for AI agents and contributors working in a project generated from Bubbleplate.

## Project Intent

This codebase started from Bubbleplate, an opinionated starter kit for Go terminal user interfaces built with the Charm stack. Treat the existing structure as the application shell for this project's real product features.

Use Bubbleplate's foundation for:

- Bubble Tea v2 application architecture
- Screen routing and navigation
- Command palette actions
- Global keybindings and help
- Header/sidebar/main/footer layout
- Centralized theme and style definitions
- In-memory debug/log screens during development
- GoReleaser-based release builds

Build the product's domain features on top of these patterns rather than bypassing them.

## Stack

- Go 1.26+
- Bubble Tea v2 via `charm.land/bubbletea/v2`
- Lip Gloss v2 via `charm.land/lipgloss/v2`
- Bubbles only where compatible with Bubble Tea v2 usage in this repo
- GoReleaser v2 config in `.goreleaser.yaml`

Be careful mixing Bubble Tea v1 and v2 types. Some Bubbles components may still return v1 `tea.Cmd` types and cannot be used directly with Bubble Tea v2 models.

## Project Layout

- `cmd/telex`: CLI entrypoint, flags, version metadata, program startup. Rename this package path and binary when adapting the template.
- `internal/app`: root Bubble Tea model, routing, global update/view wiring, keybindings, app messages.
- `internal/commands`: command registry and command palette model.
- `internal/components`: reusable shell UI pieces such as header, sidebar, footer, and modal overlays.
- `internal/debug`: in-memory event log.
- `internal/layout`: terminal size and region calculations.
- `internal/screens`: modular screens implementing the shared `Screen` interface.
- `internal/theme`: semantic theme and style definitions.

Keep package responsibilities narrow. The root app can coordinate global state, but feature-specific update and rendering should stay in dedicated screens or packages.

## Adding Product Features

When adding a first-class screen:

- Add a screen type in `internal/screens` or in a feature package that implements `screens.Screen`.
- Register it in `internal/app/app.go`.
- Add a command palette command if users should be able to open it quickly.
- Add it to sidebar navigation if it is part of the primary app flow.
- Add screen-specific keybindings through `KeyBindings()` so help stays useful.

When adding a product action:

- Prefer a command in `internal/commands` registered from `internal/app/app.go`.
- Commands should return Bubble Tea commands that emit app messages.
- Log important user-visible or debugging events through `internal/debug.Log`.

When adding layout or styling:

- Put terminal dimension calculations in `internal/layout`.
- Put reusable styles and colors in `internal/theme`.
- Prefer semantic style names over raw colors scattered through screens.

## UI Expectations

Preserve these shell behaviors unless the product explicitly requires different behavior:

- `ctrl+c` quits.
- `q` quits when no overlay is open.
- `ctrl+k` opens the command palette.
- `?` opens the help overlay.
- `esc` closes overlays.
- `tab` cycles focus.
- Sidebar navigation supports arrow keys and vim keys.
- Layout handles small terminal sizes without panics or negative dimensions.

## Template Cleanup Checklist

When turning this starter into a real project, update:

- Module path in `go.mod`.
- Binary/package path under `cmd/`.
- `project_name`, binary name, and ldflags target in `.goreleaser.yaml`.
- README title, description, development commands, and release instructions.
- GitHub repository references and release workflow assumptions.
- License owner/year if needed.
- Starter screen text that still says Bubbleplate or references template-only copy.

Do not remove the tests or release configuration unless the project has a replacement.

## Testing

Prefer tests for non-visual logic. Do not snapshot-test the full terminal UI unless specifically requested.

Before considering work complete, run:

```sh
go test ./...
```

For build verification, run:

```sh
go build ./cmd/telex
```

If the binary path has been renamed, use the project's current command path instead. Remove generated binaries after local build checks.

## Release Tooling

GoReleaser config lives in `.goreleaser.yaml` and the GitHub Actions release workflow lives in `.github/workflows/release.yml`.

If GoReleaser is installed locally, verify release packaging with:

```sh
goreleaser release --snapshot --clean
```

Keep checksum generation and platform archive coverage unless the product has a specific reason to change them.

## Coding Style

- Use `gofmt` on Go files.
- Prefer small, direct changes over large rewrites.
- Apply SOLID principles where they improve clarity and maintainability.
- Keep code DRY, but avoid premature abstractions for one-off starter or product code.
- Keep comments rare and useful.
- Default to ASCII in source and docs unless the file already uses Unicode or the UI requires it.
- Do not introduce optional dependencies unless they are used and clearly improve the product.
