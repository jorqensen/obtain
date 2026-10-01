# Introduction

`obtain` is a small command line tool that fetches the remote files you keep needing, such as an `.editorconfig`, a `LICENSE` or a CI workflow, and saves them into the current directory.

Instead of copying files from an old project or hunting for the right URL, you describe them once in a config and fetch them by name:

```sh
obtain file editorconfig
obtain pack go-project
```

## How it works

A YAML config maps short **keys** to a local filename and a remote URL. Related keys can be grouped into **packs**.

```yaml
files:
  editorconfig:
    filename: .editorconfig
    url: https://example.com/.editorconfig
  license:
    filename: LICENSE
    url: https://example.com/MIT

packs:
  go-project: [editorconfig, license]
```

- `obtain file editorconfig` downloads that one file.
- `obtain pack go-project` downloads every file in the pack.

## What you get

- **Files and packs.** Fetch one file or a whole starter set with a single command.
- **Templated downloads.** Placeholders such as the current year or your name are filled in before saving, so a license template comes out ready to use.
- **Safe by default.** Existing files are never overwritten without `--force`, and a failed download never clobbers anything.
- **A config doctor.** `obtain doctor` reports every config problem at once, with line numbers and typo suggestions.

## Where to go next

- [Installation](./installation) to get `obtain` on your machine
- [Getting started](./getting-started) for a step-by-step first run
- [Cookbook](./cookbook) for complete, copy-pasteable setups
