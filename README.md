```
 _    __
| |  / /___  __  ______ _____ ____
| | / / __ \/ / / / __ `/ __ `/ _ \
| |/ / /_/ / /_/ / /_/ / /_/ /  __/
|___/\____/\__, /\__,_/\__, /\___/
          /____/      /____/
```
_Plus loin que la nuit et le jour_


**Relational navigation CLI for Markdown notes.**

`vo <note.md>` recursively scans the note's parent directory, builds an in-memory index, resolves wikilinks, and prints outgoing links for the target note.

<img width="813" height="539" alt="image" src="https://github.com/user-attachments/assets/0516f944-f62c-473e-a71d-4858523a3ff6" />


# Install

Latest published release:

```bash
curl -fsSL https://raw.githubusercontent.com/0xJohnnyboy/voyage/main/scripts/install.sh | sh
```

Pinned version:

```bash
curl -fsSL https://raw.githubusercontent.com/0xJohnnyboy/voyage/main/scripts/install.sh | sh -s -- --version v0.1.0
```

Custom install dir:

```bash
curl -fsSL https://raw.githubusercontent.com/0xJohnnyboy/voyage/main/scripts/install.sh | sh -s -- --install-dir "$HOME/.local/bin"
```

The installer does not modify your shell profile automatically. If needed, it prints the exact
`export PATH="<dir>:$PATH"` line to add.

Uninstall:

```bash
rm -f /usr/local/bin/vo
# or, if installed elsewhere:
rm -f "$HOME/.local/bin/vo"
```

# Usage

```bash
vo [options] <path-note.md>
```

Running `vo` without arguments prints a short banner + version + compact usage. Use `-h` for full help with all options.

Options:
- `-v`, `--version` print version and exit
- `-s`, `--sort` `discovery|alpha` (default: `discovery`)
- `-f`, `--format` `simple|detailed|json` (default: `simple`)
- `-m`, `--mode` `links|tags|categories` (default: `links`)
- `-l`, `--long` alias for `--format detailed`
- `-d`, `--dangling` show unresolved links (default: `true`)
- `-D`, `--no-dangling` hide unresolved links
- `-L`, `--log-level` `silent|warn|debug` (default: `warn`)
- `-t`, `--tree` render outgoing relations as a tree (overrides list formatting)
- `-n`, `--depth` tree depth (default: `1`, valid only with `--tree`)
- `-c`, `--color` `auto|always|never` (default: `auto`)
- `--format json` is valid only with `--tree`

### Examples

```bash
vo notes/index.md
vo -s alpha notes/index.md
vo -l -D notes/index.md
vo --format detailed --dangling notes/index.md
vo --mode tags notes/index.md
vo --mode categories --tree --depth 1 notes/index.md
vo -t -n 3 notes/index.md
vo -t --long --no-dangling notes/index.md
vo -t -n 3 --format json notes/index.md
vo --color always notes/index.md
vo
```
# Related
[voyage.nvim](https://github.com/0xJohnnyboy/voyage.nvim) integrates a Telescope-like file picker with `vo` as a backend. For a lightweight zettelkasten-like note taking experience, try out [scretch.nvim](https://github.com/0xJohnnyboy/scretch.nvim).

# JSON Tree Contract (V1)

Machine-oriented JSON output is available for tree mode:

```bash
vo --format json --tree --depth <N> <path-note.md>
```

Success payload:

```json
{
  "schema_version": "1.1.0",
  "mode": "links",
  "root": {
    "id": "/abs/path/to/index.md",
    "label": "Home",
    "path": "/abs/path/to/index.md",
    "dangling": false,
    "node_kind": "note",
    "children": [
      {
        "id": "/abs/path/to/a.md",
        "label": "A",
        "path": "/abs/path/to/a.md",
        "dangling": false,
        "node_kind": "note",
        "children": []
      },
      {
        "id": "dangling:Missing Note",
        "label": "Missing Note",
        "path": "",
        "dangling": true,
        "node_kind": "note",
        "children": []
      }
    ]
  }
}
```

Node contract:
- `id` deterministic string identifier
- `label` display label
- `path` absolute path for resolved notes, empty string for dangling
- `dangling` boolean (`true` for unresolved wikilinks)
- `node_kind` one of `note|tag|category`
- `children` array of nodes

When using `--mode tags` or `--mode categories` in tree mode:
- root stays the target note
- level alternates `note -> tag/category -> note`
- `--depth 1` includes one semantic hop (`attribute + associated notes`)

Error payload (`--format json`):

```json
{
  "schema_version": "1.0.0",
  "error": {
    "code": "json_requires_tree",
    "message": "--format json is only valid with --tree"
  }
}
```

On error, Voyage returns a non-zero exit code.

# Build and Versioning

Build:

```bash
make build
```

Output binary: `dist/vo-<goos>-<goarch>` (example: `dist/vo-linux-amd64`)

Cross-platform builds:

```bash
make build-all
```

This currently generates:
- `dist/vo-linux-amd64`
- `dist/vo-darwin-amd64`
- `dist/vo-darwin-arm64`

Version:

```bash
./vo -v
./vo --version
```

Build-time versioning:
- `{tag}` when `HEAD` is exactly on a tag
- `{tag}-{short-hash}` otherwise
- `dev` when no tag exists
