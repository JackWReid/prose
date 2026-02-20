# Project Structure & Documentation Redesign

**Date:** 2026-02-20
**Status:** Approved

## Problem

All Go source files are in a flat `package main` in the root directory. No README exists for external users. Version numbers are inconsistent across files.

## Design

### Package Structure

```
prose/
  cmd/prose/
    main.go              # Arg parsing, version flag, calls editor.Run()
  internal/
    editor/              # Core editor: app, buffer, renderer, viewport, etc.
      app.go
      buffer.go
      editorbuffer.go
      renderer.go
      viewport.go
      status.go
      picker.go
      outline.go
      browser.go
      columnadjust.go
      syntax.go
      undo.go
      testdata/           # Test fixtures (test.md, test-outline.md, test_search.txt)
      *_test.go
    terminal/            # Self-contained terminal I/O abstraction
      terminal.go
      terminal_test.go
    spell/               # Spell checking
      spellcheck.go
      spellcheck_test.go
      spellcheck_integration_test.go
      dictionaries/       # Moved from root
  VERSION                # Single source of truth for version string
  Makefile
  README.md
  prose.1
  roadmap.md
```

### Dependency Graph

```
terminal  →  (nothing internal — only stdlib + x/term)
spell     →  (nothing internal — only stdlib + sajari/fuzzy)
editor    →  terminal, spell
cmd/prose →  editor
```

### Breaking the spell ↔ editor Cycle

`FindWordBoundaries(buf *Buffer)` currently lives in `spellcheck.go` but takes `*Buffer` (an editor type). This creates a circular dependency: editor needs spell types, spell needs Buffer.

**Fix:** Move `FindWordBoundaries` and `extractWordBoundariesFromLine` into the editor package. These functions perform word tokenisation, not spell checking. The spell package then has zero dependency on editor types.

### Versioning

- `VERSION` file in root contains the version string (e.g. `1.17.0`)
- `cmd/prose/main.go` declares `var Version = "dev"`
- Makefile injects version via `-ldflags "-X main.Version=$(cat VERSION)"`
- Man page version updated manually when releasing (troff files aren't worth automating)
- `go install` users get `"dev"` — standard behaviour

### README

Target audience: curious writers who may not be technical.

Sections:
1. **Title + one-liner** — "prose — a text editor for writing"
2. **Screenshot** — placeholder for a terminal screenshot
3. **What it does** — bullet points: modal editing, markdown highlighting, spell checking, distraction-free layout
4. **Installation** — `go install`, `make install`, man page note
5. **Quick start** — `prose myfile.md`, plain-English mode explanation
6. **Keybinding cheatsheet** — full table organised by mode (Default, Edit, Line-Select, Leader, Commands)
7. **Man page link** — "Run `man prose` for the full reference"

### Test Strategy

- Black-box tests (`package editor_test`, etc.) by default
- White-box tests where needed for unexported internals
- Test fixtures move to `testdata/` directories within relevant packages

### Cleanup

- Run `go mod tidy` to fix sajari/fuzzy dependency marker
- Move test fixture files to `testdata/`
- Move `dictionaries/` into `internal/spell/`
- Update `.gitignore` as needed

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Package granularity | Hybrid (3 internal packages) | Terminal is cleanly independent; spell is separable once FindWordBoundaries moves; further splitting editor would be premature abstraction for tightly coupled code |
| Cycle resolution | Move FindWordBoundaries to editor | It's tokenisation, not spell checking; simplest fix |
| Test style | Black-box default, white-box where needed | Go best practice; both are idiomatic |
| Version management | VERSION file + ldflags | Single source of truth, standard Go pattern |
| README audience | Curious writers | Project is a text editor, not a library |
