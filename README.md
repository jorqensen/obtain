# obtain

Fetch the remote files you keep needing (`.editorconfig`, a `LICENSE`, CI workflows, ...) into the current directory, using a config that maps short keys to filenames and URLs.

```
obtain file editorconfig
obtain pack go-project
```

## Documentation

Usage, configuration, variables and examples are in the docs: **https://jorqensen.github.io/obtain/**

The source is in [`docs/`](docs).

## Install

```
curl -fsSL https://raw.githubusercontent.com/jorqensen/obtain/main/install.sh | sh
```

This installs the latest release to `~/.local/bin/obtain` (Linux and macOS). Other options, including Windows, `go install` and pinning a version, are in the [installation guide](https://jorqensen.github.io/obtain/installation).

## Development

```
go build ./...
go vet ./...
go test ./...
```

The docs site needs [Bun](https://bun.sh):

```
cd docs && bun install && bun run dev
```
