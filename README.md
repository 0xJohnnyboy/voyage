# Voyage

Relational navigation CLI for Markdown notes.

`vo <note.md>` recursively scans the note's parent directory, builds an in-memory index, resolves wikilinks, and prints outgoing links for the target note.

## Usage

```bash
vo [options] <path-note.md>
```

Running `vo` without arguments shows help (`-h`) with the Voyage banner and current version.

Options:
- `-v`, `--version` print version and exit
- `-s`, `--sort` `discovery|alpha` (default: `discovery`)
- `-f`, `--format` `simple|detailed` (default: `simple`)
- `-l`, `--long` alias for `--format detailed`
- `-d`, `--dangling` show unresolved links (default: `true`)
- `-D`, `--no-dangling` hide unresolved links
- `-L`, `--log-level` `silent|warn|debug` (default: `warn`)
- `-t`, `--tree` render outgoing relations as a tree (overrides list formatting)
- `-n`, `--depth` tree depth (default: `1`, valid only with `--tree`)
- `-c`, `--color` `auto|always|never` (default: `auto`)

## Examples

```bash
vo notes/index.md
vo -s alpha notes/index.md
vo -l -D notes/index.md
vo --format detailed --dangling notes/index.md
vo -t -n 3 notes/index.md
vo -t --long --no-dangling notes/index.md
vo --color always notes/index.md
vo
```

## Install

Latest release (current tag):

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

## Build and Versioning

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
