# Telex

telex.sh CLI

## Features

- Bubble Tea v2 app shell
- Command palette
- Header/sidebar/main/footer layout
- Screen router
- Global keybindings
- Help overlay
- Theme system with Phosphor, Muted Dark, and Miami themes
- Logs/debug screen
- GoReleaser release pipeline

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

Manual publishes build and push multi-arch images to GitHub Container Registry:

```sh
docker run --rm -it ghcr.io/<owner>/telex:latest
```

## Release

```sh
git tag v0.1.0
git push origin v0.1.0
```

## Version

```sh
go run ./cmd/telex --version
```
