# Configuration

## Location

`obtain` looks for `$XDG_CONFIG_HOME/obtain/config.yaml`, falling back to `~/.config/obtain/config.yaml`. Only `.yaml` is looked up by default.

Override it with `--config` / `-c` or the `OBTAIN_CONFIG` environment variable. An explicit path is used as given, whatever its extension.

## Structure

```yaml
vars:                      # optional, see Variables
  NAME: Ada Lovelace

files:
  editorconfig:            # the key you pass to `obtain file`
    filename: .editorconfig   # required: local name
    url: https://example.com/.editorconfig   # required: remote URL
    vars: {}               # optional, per-file variables
    raw: false             # optional, skip variable rendering

packs:
  go-project: [editorconfig, license]   # keys must exist under `files`
```

### `files`

A mapping of key to entry.

| Field | Required | Description |
| --- | --- | --- |
| `filename` | yes | Name the file is saved as |
| `url` | yes | Where to download it from |
| `vars` | no | Variables for this file, overriding top-level `vars` |
| `raw` | no | `true` saves the download exactly as is, with no variable rendering |

### `packs`

A mapping of pack name to a list of file keys. `obtain pack <name>` fetches them in order and stops at the first failure. Files already fetched stay. Nothing is de-duplicated across packs.

Both `files` and `packs` are optional, and an empty config is valid.

## Validation

`obtain doctor` checks the whole config and reports every problem at once, sorted by line number:

```
config: /home/you/.config/obtain/config.yaml
  - line 14: unknown key "pack", did you mean "packs"?
  - line 20: pack "go-project" references unknown file "licence", did you mean "license"?
error: 2 problem(s) found
```

It catches:

- unknown top-level keys and unknown entry fields, with a "did you mean" suggestion for near misses
- missing or empty `filename` or `url`
- malformed `vars` (not a mapping, a bad name, a non-string value)
- wrong shapes (`files` or `packs` not a mapping, a pack not a list)
- packs that reference unknown file keys
- `vars` on a `raw: true` entry, since they would never be used

Every other command refuses to run on an invalid config and points you at `obtain doctor`. An empty config is not an error, but `doctor` notes that nothing is configured.

## How files are written

Each download goes to a temporary file in the destination directory and is then renamed into place, so a failed download never clobbers an existing file. If the destination already exists, `obtain` stops with an error instead of overwriting it, unless you pass `--force`.
