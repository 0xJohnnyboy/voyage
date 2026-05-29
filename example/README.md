# Example Zettelkasten Vault

Synthetic second-brain style notes for web/mobile development.

- 30 notes
- realistic cross-links with wikilinks
- frontmatter with `title`, `tags`, `categories`
- recurring tags and categories

## Quick Commands (by feature)

### Links (default mode)
```bash
vo example/00-index.md
vo --sort alpha example/00-index.md
vo --show path example/00-index.md
```

### Semantic Modes
```bash
vo --mode tags example/00-index.md
vo --mode categories example/00-index.md
```

### Tree and Depth
```bash
vo --tree --depth 2 example/00-index.md
vo --mode tags --tree --depth 1 example/00-index.md
```

### JSON Output
```bash
vo --tree --format json example/00-index.md
vo --mode categories --tree --depth 1 --format json example/00-index.md
```

### Scope
```bash
vo --scope up:1 example/00-index.md
vo --scope root:./example example/00-index.md
```

### Preview Note Contents
```bash
vo -D --show path example/00-index.md | xargs head -n 8
```
