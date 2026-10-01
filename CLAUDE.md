# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`obtain` is a Go CLI (urfave/cli v3) that fetches remote files you often need (dotfiles, license templates) and saves them locally. A YAML config maps a key to a filename and remote URL, and groups keys into "packs". `obtain file editorconfig` downloads the entry keyed `editorconfig`; `obtain pack go-project` downloads every file in that pack.

- Module: `github.com/jorqensen/obtain` (installable with `go install github.com/jorqensen/obtain@latest`). All internal imports use this path.
- The directory is not a git repository yet. `README.md` is repo-facing only (what it is, install, development) and links to the docs site; user-facing documentation and examples live in `docs/`, so update those on behaviour changes.

## Layout

- `main.go` — builds the `cli.Command` tree (`newApp`) and the command actions.
- `internal/config/config.go` — `Config`/`Entry` types, `DefaultPath`, `ReadFile`, `Load`, sorted key helpers.
- `internal/config/check.go` — `Check`, the config validator (see below).
- `internal/fetch/fetch.go` — `File`, the HTTP download.
- `install.sh` — POSIX sh installer (Linux/macOS) for the GitHub release archives into `~/.local/bin`. It depends on the goreleaser `name_template` (`obtain_<version>_<os>_<arch>.tar.gz`) and `checksums.txt`; change them together.
- `*_test.go` next to each of the above.
- `docs/` — VitePress site (Bun, isolated from the Go module). Pages in `docs/guide/`; keep them in sync with behaviour changes. Pages containing `{{ }}` must be wrapped in `::: v-pre`, or Vue tries to interpolate them. Build: `cd docs && bun run build`. Deployed by `.github/workflows/docs.yaml`.

## Commands

| Command | Behaviour |
| --- | --- |
| `file <key>...` | Fetch files by key. |
| `pack <name>...` | Fetch every file of each pack, in order. Stops at the first failure; files already fetched stay. No de-duplication across packs. |
| `list` (`ls`) | Table of files and packs (`text/tabwriter`); hint if nothing is configured. |
| `read` | Show the config in `$PAGER` (may include args); falls back to `less`, then plain stdout if the pager isn't installed. |
| `edit` | Open the config in `$VISUAL`, then `$EDITOR`, then `vi`. If the file is missing, creates it (and the parent dir) with the commented `config.Starter` template; an existing file is never touched. Missing-config errors point at `obtain edit`. Does not validate afterwards. |
| `doctor` | Report every problem in the config with line numbers; exit 1 if any. |

Global flags (go before the subcommand): `--config/-c` (env `OBTAIN_CONFIG`), `--dir/-d` (default `.`), `--force/-f`. Running `obtain` alone shows help; there is no root action (`obtain <key>` is not supported).

The pack command is named `pack` only. Do not add a `bundle` alias or a `bundles:` config key; there is a test guarding the alias.

## Config

Default path: `$XDG_CONFIG_HOME/obtain/config.yaml`, falling back to `~/.config/obtain/config.yaml`. There is no `.yml` fallback (`.yaml` is the recommended extension). An explicit `--config` path is used as given.

```yaml
files:
  editorconfig:
    filename: .editorconfig
    url: https://example.com/.editorconfig
packs:
  go-project: [editorconfig, license]   # keys must exist under `files`
```

Variables: downloaded files may contain `{{ obtain.NAME }}` placeholders (namespaced so they never clash with other `{{ }}` syntax). `fetch.Render` replaces them; a placeholder with no value is an error and nothing is written. Precedence: `--var/-v NAME=value`, then the entry's `vars`, then top-level `vars` (all string values). Builtins live in a separate read-only namespace, `{{ obtain.std.NAME }}` (`fetch.Builtins`: YEAR, MONTH, DAY, DATE, USER, DIRNAME); an unknown one is an error. An entry can opt out with `raw: true` (`Entry.Raw`, passed as `fetch.Vars.Raw`): the download is saved as is, and `check.go` reports `vars` on a raw entry. Rendering happens in `fetch.File` after download, so a missing variable fails only that file (a pack stops there as usual).

Format is YAML on purpose: the config is expected to grow (e.g. per-file variables/templating). New fields go on `Entry`/`Config` and must also be added to the allowed-key lists in `check.go`.

Validation (`Check`) walks the YAML nodes rather than using `Unmarshal`, so it can report all problems at once, with line numbers, sorted by line:

- unknown top-level keys and unknown entry fields, with a "did you mean" suggestion (Levenshtein distance ≤ 2, case-insensitive);
- missing/empty `filename` or `url`; malformed `vars` (top-level or per-entry: not a mapping, bad name, non-string/null value);
- wrong shapes (`files`/`packs` not mappings, a pack not a list);
- packs referencing unknown file keys (with suggestion).

Not errors: no `packs`, no `files`, an empty file. `doctor` prints "no errors, but nothing configured" for that case. `Load` (used by every command except `doctor`) fails if `Check` finds any problem and points the user at `obtain doctor`.

Fetching writes to a temp file in the destination directory and renames it, so a failed download never clobbers an existing file. Existing files are not overwritten without `--force`.

`urfave/cli-altsrc` is not used; the config is a key directory, not flag defaults.

## Development

Standard Go toolchain:

- Build: `go build ./...`
- Test all: `go test ./...`
- Single test: `go test ./internal/config -run TestCheck`
- Vet: `go vet ./...`
- Local release (no publish, output in `dist/`): `goreleaser release --snapshot --clean`. Real releases come from pushing a `v*` tag (`.github/workflows/release.yaml`). `main.version` is set via ldflags and backs `--version`.
- Try it: `go run . -c ./config.yaml list`, `go run . -c ./config.yaml file editorconfig`

## Testing

Every feature or behaviour change must come with tests, and `go test ./...` must pass before work is considered done. Conventions:

- `internal/config/config_test.go` — table-driven `Check` cases (one per validation rule, including suggestions and line numbers), `Load`, `DefaultPath`.
- `internal/fetch/fetch_test.go` — downloads against `httptest` servers; overwrite/force, bad status, no clobbering on failure.
- `main_test.go` — end-to-end command tests via `newApp().Run` with a local `httptest` server and temp dirs; the `run` helper captures stdout. Use `t.Setenv` for `PAGER`/`VISUAL`/`EDITOR`/`OBTAIN_CONFIG`.

Gotchas:

- urfave/cli calls `os.Exit` in some paths (e.g. an unknown subcommand with no root action), which kills the test binary. Don't run those through `Run`; assert on the command tree instead.
- It also treats a returned `*exec.ExitError` as an exit code and exits silently. Wrap with `%v` (not `%w`) so `main` prints the error (see `runInteractive`).
- A pager launched by `read` writes to the real terminal, not the test's captured stdout; only the stdout fallback is asserted on.
