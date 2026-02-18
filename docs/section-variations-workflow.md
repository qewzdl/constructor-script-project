# Section Variations Workflow

This project now supports a low-friction workflow for adding section variations.

## What changed

1. Section CSS files are discovered automatically from:

`themes/<theme>/static/css/sections/*.css`

`index.css` is ignored by the scanner. You no longer need to add `@import` entries manually for each new variation stylesheet.

2. A scaffold command is available to create/update section variation metadata and generate a CSS stub:

```bash
go run ./cmd/section_scaffold --section features --variation timeline --variation-label "Timeline" --variation-description "Horizontal feature timeline"
```

## Command behavior

- Updates (or creates) the section definition:
`themes/default/data/admin/sections/<section>.json`
- Adds/updates the variation metadata entry.
- Creates CSS scaffold:
`themes/default/static/css/sections/<section>-variation-<variation>.css`
- Does not overwrite existing CSS unless `--force` is passed.

## Useful flags

- `--theme` (default: `themes/default`)
- `--section` (required)
- `--variation` (required)
- `--variation-label`
- `--variation-description`
- `--section-label` (for new section files)
- `--section-description` (for new section files)
- `--order` (used for new section files, default: `100`)
- `--supports-elements` (used for new section files, default: `true`)
- `--force` (overwrite CSS file and update existing variation entry)
