# Cookbook

::: v-pre
## Bootstrap a Go project

One pack for the files every new Go repo starts with:

```yaml
files:
  editorconfig:
    filename: .editorconfig
    url: https://raw.githubusercontent.com/example/dotfiles/main/.editorconfig
  gitignore:
    filename: .gitignore
    url: https://raw.githubusercontent.com/example/dotfiles/main/go.gitignore
  license:
    filename: LICENSE
    url: https://raw.githubusercontent.com/example/dotfiles/main/MIT

packs:
  go-project: [editorconfig, gitignore, license]
```

```sh
mkdir my-project && obtain -d my-project pack go-project
```

## A license with your name and the current year

Host a template that uses placeholders:

```text
MIT License

Copyright (c) {{ obtain.std.YEAR }} {{ obtain.NAME }}
```

Set your name once at the top level of the config:

```yaml
vars:
  NAME: Ada Lovelace

files:
  license:
    filename: LICENSE
    url: https://example.com/templates/MIT
```

```sh
obtain file license
```

## Override a variable for one run

```sh
obtain -f -v NAME="Grace Hopper" file license
```

`-f` is needed because `LICENSE` already exists.

## Name a project after its directory

`DIRNAME` is the name of the directory being saved into, which is handy for READMEs and module files:

```text
# {{ obtain.std.DIRNAME }}
```

```sh
obtain -d ./my-project file readme
```

## CI workflows that contain `${{ }}`

GitHub Actions syntax like `${{ secrets.TOKEN }}` doesn't use the `obtain.` namespace, so it passes through untouched. No `raw` needed:

```yaml
files:
  ci:
    filename: .github/workflows/ci.yaml
    url: https://example.com/templates/ci.yaml
```

## Use a project-local config

```sh
obtain -c ./obtain.yaml list
export OBTAIN_CONFIG=./obtain.yaml   # or set it for the whole shell
```

## Scaffold a whole repository in one command

`obtain` creates the destination directory, and any parent directories of a file's `filename`, so you can start from nothing. A `service` pack that covers the usual starting files:

```yaml
vars:
  NAME: Ada Lovelace

files:
  readme:
    filename: README.md
    url: https://example.com/templates/README.md      # starts with "# {{ obtain.std.DIRNAME }}"
  license:
    filename: LICENSE
    url: https://example.com/templates/MIT            # "Copyright (c) {{ obtain.std.YEAR }} {{ obtain.NAME }}"
  gitignore:
    filename: .gitignore
    url: https://example.com/templates/go.gitignore
  ci:
    filename: .github/workflows/ci.yaml
    url: https://example.com/templates/ci.yaml

packs:
  service: [readme, license, gitignore, ci]
```

```sh
obtain -d billing-api pack service
```

`billing-api/` is created, the README gets its title from the directory name, and `.github/workflows/` appears without a `mkdir`.

## Wrap it in a shell function

Because `-d` creates the directory and `DIRNAME` follows it, a tiny function gives you a "new project" command:

```sh
# bash / zsh
newgo() { obtain -d "$1" pack service && cd "$1" && go mod init "example.com/$1"; }
```

```fish
# fish
function newgo
    obtain -d $argv[1] pack service; and cd $argv[1]; and go mod init example.com/$argv[1]
end
```

## Pin a template to a commit

A URL pointing at `main` changes whenever the template repository does. To get the same file every time, pin the URL to a commit or tag:

```yaml
files:
  ci:
    filename: .github/workflows/ci.yaml
    url: https://raw.githubusercontent.com/example/templates/3f9c2ab/ci.yaml
```

Bump the hash when you want to adopt a newer version.

## One template, several files

Two entries can share a URL and differ only in `filename` and `vars`. The template is downloaded once per entry and rendered with each entry's own variables:

```yaml
files:
  dockerfile-api:
    filename: services/api/Dockerfile
    url: https://example.com/templates/Dockerfile
    vars:
      SERVICE: api
      PORT: "8080"
  dockerfile-worker:
    filename: services/worker/Dockerfile
    url: https://example.com/templates/Dockerfile
    vars:
      SERVICE: worker
      PORT: "9090"

packs:
  services: [dockerfile-api, dockerfile-worker]
```

Values must be strings, so quote numbers like `"8080"`.

## Date-stamped notes

`DATE` is handy for files that record when they were created, such as an architecture decision record:

```text
# ADR: <title>

Date: {{ obtain.std.DATE }}
Author: {{ obtain.std.USER }}
Status: proposed
```

```sh
obtain -d docs/adr file adr
```

## Refresh a file across many repositories

Each `obtain` run is independent, so a shell loop updates a shared file everywhere. `--force` is required to overwrite the existing copy:

```sh
for repo in ~/code/*/; do
  obtain -d "$repo" -f file ci
done
```

Review the result with `git diff` in each repository before committing. A failure in one repository doesn't stop the loop.

## Try a pack without touching your project

Save into a scratch directory to see what a pack produces, including how variables render:

```sh
obtain -d "$(mktemp -d)" pack service
```

The command prints where each file went.

## Keep separate configs for work and personal use

Point `OBTAIN_CONFIG` at a different file per context. For example with [direnv](https://direnv.net), in a work directory's `.envrc`:

```sh
export OBTAIN_CONFIG=$HOME/.config/obtain/work.yaml
```

Everything under that directory now uses the work config, and the default config stays untouched elsewhere.

## Version your config and check it in CI

The config is plain YAML, so it fits in a dotfiles or templates repository. `obtain doctor` exits non-zero on any problem, which makes it a one-line CI check:

```yaml
- run: obtain -c obtain.yaml doctor
```

It catches typos in keys, pack entries that point at missing files and malformed `vars` before they reach a machine.
:::
