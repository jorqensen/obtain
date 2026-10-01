# Commands

| Command | Description |
| --- | --- |
| `obtain file <key>...` | Fetch one or more files by key |
| `obtain pack <name>...` | Fetch every file in one or more packs, in order |
| `obtain list` (`ls`) | Show configured files and packs |
| `obtain read` | Show the config file in `$PAGER` |
| `obtain edit` | Open the config file in `$VISUAL` / `$EDITOR` |
| `obtain doctor` | Check the config and report every problem |

Running `obtain` on its own shows help. `obtain <key>` is not supported; use `obtain file <key>`.

## Global flags

| Flag | Description |
| --- | --- |
| `--config`, `-c` | Path to the config file (env: `OBTAIN_CONFIG`) |
| `--dir`, `-d` | Directory to save files into (default: `.`) |
| `--var`, `-v` | Set a variable, `NAME=value` (repeatable) |
| `--force`, `-f` | Overwrite existing files (by default an existing file is an error) |

Global flags go before the subcommand:

```sh
obtain -d ./out --force pack go-project
```

## `file`

```sh
obtain file <key>...
```

Fetches one or more files by the keys under `files` in your config, in the order given. Each saved file is reported as `key -> path`.

```sh
obtain file editorconfig license
obtain -d ./my-project file license
```

- It stops at the first failure: a download error, a missing variable, an unknown key or a file that already exists (unless you pass `--force`). Files fetched before that stay in place.
- Keys are looked up as it goes, so a typo in the second key is only reported after the first file has been fetched. Run `obtain doctor` or `obtain list` if you're unsure.
- With no arguments it shows help.

## `pack`

```sh
obtain pack <name>...
```

Fetches every file in each named pack, in the order the pack lists them. It works like `file` for each key.

```sh
obtain pack go-project
obtain pack go-project docs-project
```

- Pack names are checked before anything is downloaded, so an unknown pack fails without writing any files.
- It stops at the first failed download. Files already fetched stay in place.
- Nothing is de-duplicated across packs. If two packs share a key, that file is fetched twice, and the second attempt fails on the existing file unless you pass `--force`.
- With no arguments it shows help.

## `list`

Prints a table of files and packs. If nothing is configured, it prints a hint instead.

## `read`

Shows the config in `$PAGER` (which may include arguments). Falls back to `less`, then to plain stdout if the pager isn't installed.

## `edit`

Opens the config in `$VISUAL`, then `$EDITOR`, then `vi`. If the file is missing, it is created (along with its parent directory) from a commented starter template. An existing file is never touched. `edit` does not validate afterwards, so run `obtain doctor`.

## `doctor`

Reports every problem in the config with line numbers and exits 1 if there are any. See [Configuration](./config#validation).
