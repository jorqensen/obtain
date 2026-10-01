# Getting started

New to `obtain`? This walks you through your first config and your first download. If you haven't installed it yet, start with [Installation](./installation).

## Create a config

```sh
obtain edit
```

This opens the config in `$VISUAL`, then `$EDITOR`, then `vi`. If the file doesn't exist yet it is created from a commented starter template. An existing config is never touched.

The default location is `$XDG_CONFIG_HOME/obtain/config.yaml`, falling back to `~/.config/obtain/config.yaml`.

Add a file and a pack:

```yaml [config.yaml]
files:
  editorconfig:
    filename: .editorconfig
    url: https://raw.githubusercontent.com/example/dotfiles/main/.editorconfig
  license:
    filename: LICENSE
    url: https://raw.githubusercontent.com/example/dotfiles/main/LICENSE

packs:
  go-project: [editorconfig, license]
```

Check it:

```sh
obtain doctor
```

## Fetch files

```sh
obtain file editorconfig        # one file
obtain file editorconfig license
obtain pack go-project          # every file in the pack
```

Files land in the current directory. Use `--dir` to save elsewhere:

```sh
obtain --dir ./my-project pack go-project
```

An existing file is never overwritten: `obtain` stops with an error. Pass `--force` to overwrite.

::: tip
Global flags go **before** the subcommand: `obtain -d ./out --force pack go-project`.
:::

## Next steps

- [Configuration](./config) for every field and how validation works
- [Variables](./variables) to template downloaded files
- [Cookbook](./cookbook) for complete, copy-pasteable setups
