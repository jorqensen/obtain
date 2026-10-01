# Variables

::: v-pre
Remote files can contain placeholders that `obtain` fills in before saving. There are two kinds, both namespaced with `obtain.` so they never clash with other `{{ }}` syntax. Anything else, such as a GitHub Actions `${{ secrets.X }}`, is left untouched.

| Placeholder | Source |
| --- | --- |
| `{{ obtain.NAME }}` | Your own variable, from `--var` or `vars:` in the config |
| `{{ obtain.std.NAME }}` | A builtin computed by `obtain` |

Whitespace inside the braces is optional: `{{obtain.NAME}}` works too.

## Example

A `LICENSE` template hosted at `https://example.com/templates/MIT`:

```text
MIT License

Copyright (c) {{ obtain.std.YEAR }} {{ obtain.NAME }}
...
```

Config:

```yaml
vars:
  NAME: Ada Lovelace

files:
  license:
    filename: LICENSE
    url: https://example.com/templates/MIT

  readme:
    filename: README.md
    url: https://example.com/templates/README.md
    vars:
      NAME: Ada's Team
```

```sh
$ obtain file license
$ head -3 LICENSE
MIT License

Copyright (c) 2026 Ada Lovelace

$ obtain -f -v NAME="Grace Hopper" file license   # override for one run
```

## Precedence

Values for `{{ obtain.NAME }}` are resolved in this order, first match wins:

1. `--var` / `-v NAME=value` (repeatable, placed before the subcommand)
2. `vars:` on the file entry
3. top-level `vars:`

```sh
obtain -v NAME=Grace -v EMAIL=grace@example.com pack new-project
```

A placeholder with no value is an error, and nothing is written for that file. In a pack, the run stops there as usual.

Variable names use letters, digits and `_`. Values must be strings. Files without placeholders are saved byte for byte.

## Builtins

`{{ obtain.std.NAME }}` values are computed by `obtain` and cannot be overridden by `--var` or `vars:`.

| Name | Value | Example |
| --- | --- | --- |
| `YEAR` | Current year | `2026` |
| `MONTH` | Current month, zero-padded | `03` |
| `DAY` | Current day of month, zero-padded | `09` |
| `DATE` | Current date, `YYYY-MM-DD` | `2026-03-09` |
| `USER` | OS username of whoever runs `obtain` | `ada` |
| `DIRNAME` | Name of the directory being saved into (`--dir`) | `my-project` |

An unknown builtin is an error that lists the available ones.

## Opting out with `raw`

If a file legitimately contains `{{ obtain.NAME }}` (for example, a template for another tool that uses the same syntax), set `raw: true` on its entry:

```yaml
files:
  template:
    filename: template.txt
    url: https://example.com/template.txt
    raw: true
```

It is saved exactly as downloaded, and `--var` has no effect on it. `obtain doctor` reports `vars:` on a raw entry, since they would never be used.
:::
