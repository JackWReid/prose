# Project Structure & Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the flat `package main` Go project into `cmd/` + `internal/` packages, add versioning, and write a user-facing README.

**Architecture:** Three internal packages (`editor`, `terminal`, `spell`) with a thin `cmd/prose/main.go` entry point. Terminal is a self-contained leaf; spell has no internal deps; editor imports both. Version injected via ldflags at build time.

**Tech Stack:** Go 1.25, `golang.org/x/term`, `github.com/sajari/fuzzy`, `go:embed`

**Reference:** See `docs/plans/2026-02-20-project-structure-design.md` for the approved design.

---

### Task 1: Create directory structure

**Files:**
- Create: `cmd/prose/` (directory)
- Create: `internal/editor/` (directory)
- Create: `internal/terminal/` (directory)
- Create: `internal/spell/` (directory)

**Step 1: Create all directories**

```bash
mkdir -p cmd/prose internal/editor internal/terminal internal/spell
```

**Step 2: Commit**

```bash
git add cmd internal
git commit -m "chore: create package directory structure"
```

---

### Task 2: Extract terminal package

The terminal package is completely self-contained — it only imports stdlib and `golang.org/x/term`. This is the cleanest extraction.

**Files:**
- Move: `terminal.go` → `internal/terminal/terminal.go`
- Move: `terminal_test.go` → `internal/terminal/terminal_test.go`

**Step 1: Move files**

```bash
git mv terminal.go internal/terminal/terminal.go
git mv terminal_test.go internal/terminal/terminal_test.go
```

**Step 2: Update package declaration**

In `internal/terminal/terminal.go`, change line 1:
```go
package terminal
```

In `internal/terminal/terminal_test.go`, change line 1:
```go
package terminal
```

All exported types (`Terminal`, `Key`, `InputEvent`, `MouseEvent`, `MouseButton`) and constants (`KeyRune`, `KeyEscape`, etc., `EventKey`, `EventMouse`, `MouseLeft`, etc.) are already capitalised and exported. The unexported helpers (`parseKey`, `parseInput`, `parseMouseEvent`, `decodeUTF8`) are tested via white-box tests (same package), which is correct since they're implementation details.

**Step 3: Verify terminal package compiles and tests pass**

```bash
cd internal/terminal && go build ./... && go test -v ./...
```

Expected: all 18 terminal tests pass.

**Step 4: Commit**

```bash
git add internal/terminal/
git commit -m "refactor: extract internal/terminal package"
```

---

### Task 3: Prepare spell extraction — move word boundary functions

`FindWordBoundaries` and `extractWordBoundariesFromLine` use `*Buffer` (an editor type), creating a circular dependency if left in spell. They perform word tokenisation, not spell checking, so they belong in editor.

`WordBoundary` is the return type of these functions and is used in `app.go` for word jumping navigation. It also belongs in editor.

**Files:**
- Create: `internal/editor/wordboundaries.go`
- Create: `internal/editor/wordboundaries_test.go`

**Step 1: Create `internal/editor/wordboundaries.go`**

Extract from `spellcheck.go` (lines 22-27, 163-220):

```go
package editor

import "unicode"

// WordBoundary represents a word location in the buffer for navigation.
type WordBoundary struct {
	Line     int
	StartCol int
	EndCol   int
}

// FindWordBoundaries scans the entire buffer and returns all word boundaries.
// Words are defined using Vim-style definition: sequences of letters, digits, and underscores.
func FindWordBoundaries(buf *Buffer) []WordBoundary {
	var boundaries []WordBoundary

	for lineNum := 0; lineNum < len(buf.Lines); lineNum++ {
		line := buf.Lines[lineNum]
		lineBoundaries := extractWordBoundariesFromLine(lineNum, line)
		boundaries = append(boundaries, lineBoundaries...)
	}

	return boundaries
}

// extractWordBoundariesFromLine finds all word boundaries in a single line.
func extractWordBoundariesFromLine(lineNum int, line string) []WordBoundary {
	var boundaries []WordBoundary
	runes := []rune(line)

	inWord := false
	var startCol int

	for i, r := range runes {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'

		if isWordChar {
			if !inWord {
				startCol = i
				inWord = true
			}
		} else {
			if inWord {
				boundaries = append(boundaries, WordBoundary{
					Line:     lineNum,
					StartCol: startCol,
					EndCol:   i,
				})
				inWord = false
			}
		}
	}

	if inWord {
		boundaries = append(boundaries, WordBoundary{
			Line:     lineNum,
			StartCol: startCol,
			EndCol:   len(runes),
		})
	}

	return boundaries
}
```

**Step 2: Create `internal/editor/wordboundaries_test.go`**

Extract from `spellcheck_test.go` (lines 240-390) — the `TestExtractWordBoundariesFromLine` and `TestFindWordBoundaries` tests. Change package to `editor`.

```go
package editor

import "testing"

func TestExtractWordBoundariesFromLine(t *testing.T) {
	// ... (copy exact test table from spellcheck_test.go lines 241-351)
}

func TestFindWordBoundaries(t *testing.T) {
	// ... (copy exact test table from spellcheck_test.go lines 354-390)
}
```

**Step 3: Remove the moved code from root files**

In `spellcheck.go`: delete `WordBoundary` struct (lines 22-27), `FindWordBoundaries` (lines 163-175), and `extractWordBoundariesFromLine` (lines 177-220).

In `spellcheck_test.go`: delete `TestExtractWordBoundariesFromLine` (lines 240-352) and `TestFindWordBoundaries` (lines 354-390).

**Step 4: Verify editor package word boundary tests pass**

```bash
cd internal/editor && go test -v -run "WordBoundaries" ./...
```

Expected: both word boundary tests pass.

**Step 5: Verify root spellcheck tests still pass**

```bash
go test -v -run "Spell|Extract" ./...
```

Expected: remaining spell tests pass (TestNewSpellChecker, TestCheckWord, TestExtractWords, TestCheckLine, TestCheckLinePositions). Root code still compiles because `FindWordBoundaries` callers in app.go aren't moved yet.

Note: the root won't compile at this point because `WordBoundary` is still referenced in `app.go`. We'll fix that in Task 5 when everything moves.

Actually — since we're removing `WordBoundary` from the root, `app.go` will break. So instead: **keep `WordBoundary` in root spellcheck.go for now**. Only move `FindWordBoundaries` and `extractWordBoundariesFromLine` to internal/editor, and **also define a duplicate `WordBoundary` in internal/editor/wordboundaries.go**. The root copy gets deleted in Task 5 when app.go moves to internal/editor.

Revised Step 3: In `spellcheck.go`, delete only `FindWordBoundaries` (lines 163-175) and `extractWordBoundariesFromLine` (lines 177-220). Keep `WordBoundary` struct (lines 22-27) in place.

**Step 5 (revised): Verify root still compiles**

```bash
go build ./...
```

Expected: compiles. `app.go` still references `WordBoundary` from root's spellcheck.go. `FindWordBoundaries` calls in app.go will fail — so actually we need to keep `FindWordBoundaries` in the root too for now, or move app.go at the same time.

**REVISED APPROACH:** Skip this task as a standalone step. The word boundary extraction happens simultaneously with the big move in Task 5. It's not possible to do it incrementally due to Go's single-package-per-directory constraint.

**Step 1: Just create the files for later**

Create `internal/editor/wordboundaries.go` and `internal/editor/wordboundaries_test.go` with the code above. They'll compile in isolation within the `editor` package. Don't modify root files yet.

**Step 2: Verify they compile**

```bash
cd internal/editor && go vet ./...
```

Expected: passes (the file references `Buffer` which doesn't exist yet, so this will fail in isolation).

**FURTHER REVISED:** Write the file but don't verify until Task 5. Commit as part of the big restructure.

---

### Task 4: Extract spell package

**Files:**
- Move: `spellcheck.go` → `internal/spell/spellcheck.go`
- Move: `spellcheck_test.go` → `internal/spell/spellcheck_test.go`
- Move: `dictionaries/` → `internal/spell/dictionaries/`

**Step 1: Move files**

```bash
git mv dictionaries internal/spell/dictionaries
```

Create `internal/spell/spellcheck.go` from current `spellcheck.go` with these changes:
- Package declaration: `package spell`
- Remove `WordBoundary`, `FindWordBoundaries`, `extractWordBoundariesFromLine` (they go to editor)
- Update embed path: `//go:embed dictionaries/en_GB-large.txt` (stays the same since dictionaries moves with it)
- All exported types (`SpellChecker`, `SpellError`, `NewSpellChecker`, `CheckWord`, `ExtractWords`, `CheckLine`) keep their names
- `wordPosition` stays unexported — it's internal to spell

Create `internal/spell/spellcheck_test.go` from current `spellcheck_test.go` with:
- Package declaration: `package spell`
- Remove `TestExtractWordBoundariesFromLine` and `TestFindWordBoundaries` (they go to editor)
- Keep: `TestNewSpellChecker`, `TestCheckWord`, `TestExtractWords`, `TestCheckLine`, `TestCheckLinePositions`
- Tests reference `sc.model` (unexported field) — white-box testing, which is fine since same package

**Step 2: Verify spell package compiles and tests pass**

```bash
cd internal/spell && go test -v ./...
```

Expected: 5 spell tests pass.

**Step 3: Don't delete root files yet** — they're needed until Task 5.

---

### Task 5: Move editor files and create cmd/prose entry point

This is the big coordinated move. All remaining root `.go` files move to `internal/editor/`, and a new `cmd/prose/main.go` is created.

**Files:**
- Move: `app.go` → `internal/editor/app.go`
- Move: `buffer.go` → `internal/editor/buffer.go`
- Move: `editorbuffer.go` → `internal/editor/editorbuffer.go`
- Move: `renderer.go` → `internal/editor/renderer.go`
- Move: `viewport.go` → `internal/editor/viewport.go`
- Move: `status.go` → `internal/editor/status.go`
- Move: `picker.go` → `internal/editor/picker.go`
- Move: `outline.go` → `internal/editor/outline.go`
- Move: `browser.go` → `internal/editor/browser.go`
- Move: `columnadjust.go` → `internal/editor/columnadjust.go`
- Move: `syntax.go` → `internal/editor/syntax.go`
- Move: `undo.go` → `internal/editor/undo.go`
- Move: all `*_test.go` files → `internal/editor/`
- Create: `internal/editor/wordboundaries.go` (from Task 3)
- Create: `internal/editor/wordboundaries_test.go` (from Task 3)
- Create: `cmd/prose/main.go`
- Delete: root `main.go`, root `spellcheck.go`, root `spellcheck_test.go`

**Step 1: Move all source files**

```bash
# Editor source files
git mv app.go internal/editor/app.go
git mv buffer.go internal/editor/buffer.go
git mv editorbuffer.go internal/editor/editorbuffer.go
git mv renderer.go internal/editor/renderer.go
git mv viewport.go internal/editor/viewport.go
git mv status.go internal/editor/status.go
git mv picker.go internal/editor/picker.go
git mv outline.go internal/editor/outline.go
git mv browser.go internal/editor/browser.go
git mv columnadjust.go internal/editor/columnadjust.go
git mv syntax.go internal/editor/syntax.go
git mv undo.go internal/editor/undo.go

# Test files
git mv buffer_test.go internal/editor/buffer_test.go
git mv editorbuffer_test.go internal/editor/editorbuffer_test.go
git mv renderer_test.go internal/editor/renderer_test.go
git mv viewport_test.go internal/editor/viewport_test.go
git mv status_test.go internal/editor/status_test.go
git mv picker_test.go internal/editor/picker_test.go
git mv command_test.go internal/editor/command_test.go
git mv syntax_test.go internal/editor/syntax_test.go
git mv undo_test.go internal/editor/undo_test.go
git mv browser_test.go internal/editor/browser_test.go
git mv columnadjust_test.go internal/editor/columnadjust_test.go
git mv navigation_integration_test.go internal/editor/navigation_integration_test.go
git mv spellcheck_integration_test.go internal/editor/spellcheck_integration_test.go

# Delete root copies of files now in spell package
git rm spellcheck.go
git rm spellcheck_test.go
git rm main.go
```

**Step 2: Update all package declarations**

In every file under `internal/editor/`, change:
```go
package main
```
to:
```go
package editor
```

**Step 3: Add imports for terminal and spell packages**

In `internal/editor/app.go`, add to imports:
```go
"github.com/JackWReid/prose/internal/terminal"
"github.com/JackWReid/prose/internal/spell"
```

In `internal/editor/editorbuffer.go`, add to imports:
```go
"github.com/JackWReid/prose/internal/spell"
```

In `internal/editor/renderer.go`, add to imports:
```go
"github.com/JackWReid/prose/internal/spell"
```

In `internal/editor/status.go`, add to imports:
```go
"github.com/JackWReid/prose/internal/terminal"
```

In `internal/editor/spellcheck_integration_test.go`, add to imports:
```go
"github.com/JackWReid/prose/internal/spell"
```

**Step 4: Update type references throughout editor package**

All references to terminal types need the `terminal.` prefix:
- `*Terminal` → `*terminal.Terminal`
- `NewTerminal()` → `terminal.NewTerminal()`
- `Key` (as a type) → `terminal.Key`
- `InputEvent` → `terminal.InputEvent`
- `MouseEvent` → `terminal.MouseEvent`
- `MouseButton` → `terminal.MouseButton`
- `EventKey` → `terminal.EventKey`
- `EventMouse` → `terminal.EventMouse`
- `KeyRune` → `terminal.KeyRune`
- `KeyEscape` → `terminal.KeyEscape`
- `KeyEnter` → `terminal.KeyEnter`
- `KeyBackspace` → `terminal.KeyBackspace`
- `KeyUp` → `terminal.KeyUp`
- `KeyDown` → `terminal.KeyDown`
- `KeyLeft` → `terminal.KeyLeft`
- `KeyRight` → `terminal.KeyRight`
- `KeyCtrlZ` → `terminal.KeyCtrlZ`
- `KeyCtrlY` → `terminal.KeyCtrlY`
- `KeyCtrlR` → `terminal.KeyCtrlR`
- `KeyCtrlD` → `terminal.KeyCtrlD`
- `KeyCtrlU` → `terminal.KeyCtrlU`
- `KeyHome` → `terminal.KeyHome`
- `KeyEnd` → `terminal.KeyEnd`
- `KeyDelete` → `terminal.KeyDelete`
- `KeyPgUp` → `terminal.KeyPgUp`
- `KeyPgDn` → `terminal.KeyPgDn`
- `KeyUnknown` → `terminal.KeyUnknown`
- `MouseLeft` → `terminal.MouseLeft`

All references to spell types need the `spell.` prefix:
- `*SpellChecker` → `*spell.SpellChecker`
- `NewSpellChecker()` → `spell.NewSpellChecker()`
- `SpellError` → `spell.SpellError`
- `[]SpellError` → `[]spell.SpellError`

**Files requiring terminal prefix changes:**
- `app.go`: extensive — `Terminal` field, all key handling functions, event types
- `status.go`: `HandlePromptKey(key Key)` → `HandlePromptKey(key terminal.Key)`, key constants

**Files requiring spell prefix changes:**
- `app.go`: `SpellChecker` field, `NewSpellChecker()` call, `SpellError` field reads
- `editorbuffer.go`: `[]SpellError` field, `*SpellChecker` parameter
- `renderer.go`: `[]SpellError` parameter in `RenderFrame` and `applySpellHighlighting`
- `spellcheck_integration_test.go`: `NewSpellChecker()`, `SpellError`, `CheckLine`

**Step 5: Export `App`, `NewApp`, `Run` for external access**

These are already capitalised and exported. No changes needed.

**Step 6: Create `internal/editor/wordboundaries.go`**

The `WordBoundary` struct, `FindWordBoundaries`, and `extractWordBoundariesFromLine` as described in Task 3 above. These are now in `package editor` and can reference `*Buffer` directly.

**Step 7: Create `internal/editor/wordboundaries_test.go`**

The `TestExtractWordBoundariesFromLine` and `TestFindWordBoundaries` tests from Task 3.

**Step 8: Create `cmd/prose/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/JackWReid/prose/internal/editor"
)

var Version = "dev"

func main() {
	filenames := os.Args[1:]

	app := editor.NewApp(filenames)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prose: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 9: Verify compilation**

```bash
go build ./cmd/prose/
```

Fix any remaining compilation errors. Common issues:
- Missing imports
- Unexported fields accessed across package boundaries (there shouldn't be any since editor tests are in `package editor`)
- Embed path issues with dictionaries

**Step 10: Run all tests**

```bash
go test ./...
```

Expected: all tests pass across all three packages.

**Step 11: Verify the binary works**

```bash
go run ./cmd/prose/ test.md
```

Open a file, navigate around, quit with `:q`. Verify basic functionality.

**Step 12: Commit**

```bash
git add -A
git commit -m "refactor: restructure into cmd/ + internal/ packages

Move editor code into internal/editor, terminal I/O into internal/terminal,
and spell checking into internal/spell. Entry point is now cmd/prose/main.go.

Dependency graph: editor → terminal, spell. No circular dependencies."
```

---

### Task 6: Add VERSION file and update Makefile

**Files:**
- Create: `VERSION`
- Modify: `Makefile`
- Modify: `cmd/prose/main.go`
- Modify: `prose.1` (line 1)

**Step 1: Create VERSION file**

```
1.17.0
```

(Latest version based on roadmap — 1.17 is the most recent feature release.)

**Step 2: Update Makefile**

```makefile
# prose Makefile

# Installation directories
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1

# Binary name
BINARY := prose

# Read version from VERSION file
VERSION := $(shell cat VERSION)

# Build the prose binary
build:
	@echo "Building $(BINARY) v$(VERSION)..."
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/prose/
	@echo "Build complete: $(BINARY)"

# Install prose binary and man page
install: build
	@echo "Installing $(BINARY) to $(DESTDIR)$(BINDIR)..."
	mkdir -p $(DESTDIR)$(BINDIR)
	mkdir -p $(DESTDIR)$(MANDIR)
	install -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -m 0644 prose.1 $(DESTDIR)$(MANDIR)/prose.1
	@echo "Installation complete!"
	@echo "Binary: $(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "Man page: $(DESTDIR)$(MANDIR)/prose.1"

# Install just the man page (useful when binary is installed via go install)
install-man:
	@echo "Installing man page to $(DESTDIR)$(MANDIR)..."
	mkdir -p $(DESTDIR)$(MANDIR)
	install -m 0644 prose.1 $(DESTDIR)$(MANDIR)/prose.1
	@echo "Man page installed: $(DESTDIR)$(MANDIR)/prose.1"

# Uninstall prose binary and man page
uninstall:
	@echo "Uninstalling $(BINARY)..."
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -f $(DESTDIR)$(MANDIR)/prose.1
	@echo "Uninstall complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build and run
run: build
	./$(BINARY)

# Show help
help:
	@echo "prose Makefile targets:"
	@echo "  make build        - Build the prose binary"
	@echo "  make install      - Install prose binary and man page (default PREFIX=/usr/local)"
	@echo "  make install-man  - Install just the man page (for use with go install)"
	@echo "  make uninstall    - Remove installed prose binary and man page"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run all tests"
	@echo "  make run          - Build and run prose"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX            - Installation prefix (default: /usr/local)"
	@echo "  BINDIR            - Binary installation directory (default: PREFIX/bin)"
	@echo "  MANDIR            - Man page installation directory (default: PREFIX/share/man/man1)"
	@echo "  DESTDIR           - Staging directory for package builds"
	@echo ""
	@echo "Examples:"
	@echo "  make install PREFIX=/usr/local"
	@echo "  make install PREFIX=~/.local"
	@echo "  make install DESTDIR=/tmp/staging PREFIX=/usr"

.PHONY: build install install-man uninstall clean test run help
```

Key changes:
- `VERSION` variable read from file
- Build command now uses `./cmd/prose/` and injects version via ldflags
- Version shown in build output

**Step 3: Update man page version**

In `prose.1` line 1, change:
```
.TH PROSE 1 "February 2026" "prose 1.11.0" "User Commands"
```
to:
```
.TH PROSE 1 "February 2026" "prose 1.17.0" "User Commands"
```

**Step 4: Verify version injection works**

```bash
make build
./prose --version 2>/dev/null || echo "No --version flag yet — verify by checking binary"
```

Note: there's no `--version` flag in the current code. That's fine — the version is embedded in the binary and could be added later.

**Step 5: Verify tests pass**

```bash
make test
```

**Step 6: Commit**

```bash
git add VERSION Makefile cmd/prose/main.go prose.1
git commit -m "feat: add VERSION file and build-time version injection"
```

---

### Task 7: Write README

**Files:**
- Create: `README.md`

**Step 1: Read man page for comprehensive keybinding reference**

Read `prose.1` thoroughly to extract all keybindings for the cheatsheet.

**Step 2: Write README.md**

Structure:
1. Title + one-liner
2. Screenshot placeholder
3. What it does (4 bullet points)
4. Installation (go install, make install, man page)
5. Quick start (modes explained in plain English)
6. Keybinding cheatsheet (tables by mode)
7. Man page link

Target audience: curious writers. British spelling. No jargon where avoidable. Explain vim concepts briefly for non-vim users.

The keybinding cheatsheet should cover:
- **Default mode:** movement (h/j/k/l, w/b, gg/G, 0/$, Ctrl-D/U, PgUp/PgDn), editing (dd, yy, p/P, x/X), mode switching (i, o, O, A, V), search (/), spell navigation (x/X when spellcheck on)
- **Edit mode:** typing, Esc to return, Backspace, Delete, Enter, Home/End, arrows
- **Line-Select mode:** V to enter, j/k to extend, y to yank, d to delete, s to send to scratch
- **Leader commands (Space+):** H (outline), O (browser), - (column width)
- **Command mode (:):** w, q, wq, q!, qa, wqa, spell, rename

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add user-facing README with keybinding cheatsheet"
```

---

### Task 8: Final cleanup

**Files:**
- Modify: `.gitignore`
- Modify: `go.mod` (via go mod tidy)

**Step 1: Run go mod tidy**

```bash
go mod tidy
```

This fixes the `github.com/sajari/fuzzy` dependency marker flagged in diagnostics.

**Step 2: Update .gitignore**

```
.DS_Store
.task/
prose
debug_screenshots/
```

No changes needed — the binary name `prose` is already ignored. The `debug_screenshots/` directory is already ignored.

**Step 3: Clean up test fixture files**

The files `test.md`, `test-outline.md`, `test_search.txt` are manual test files not referenced by any Go test code. Two options:
- Leave them at root (convenient for `./prose test.md` during development)
- Move to a `testdata/` dir

Recommendation: leave at root. They're dev conveniences, not test fixtures.

**Step 4: Verify everything**

```bash
make clean && make build && make test
```

Expected: clean build, all tests pass.

**Step 5: Commit**

```bash
git add go.mod go.sum .gitignore
git commit -m "chore: run go mod tidy and clean up"
```

---

## Summary of commits

1. `chore: create package directory structure`
2. `refactor: extract internal/terminal package`
3. `refactor: restructure into cmd/ + internal/ packages`
4. `feat: add VERSION file and build-time version injection`
5. `docs: add user-facing README with keybinding cheatsheet`
6. `chore: run go mod tidy and clean up`

## Notes for implementer

- **Go's compilation constraint:** all `.go` files in a directory must share the same package. You cannot incrementally move one file at a time and have the project compile between moves. The terminal extraction (Task 2) works because terminal.go has no dependencies on other project files. The big move (Task 5) must happen as one coordinated change.

- **The `terminal.` prefix change is the most tedious part.** There are ~100+ references to terminal types and constants throughout `app.go` and `status.go`. Use find-and-replace carefully. Do `Key` → `terminal.Key` LAST (after `KeyRune`, `KeyEscape` etc.) to avoid double-prefixing.

- **Test files that access unexported fields** (like `sc.model` in spell tests, or `eb.cursorLine` in editor tests) must stay in the same package as the code they test. This is white-box testing, which is idiomatic Go.

- **The spellcheck_integration_test.go stays in the editor package** because it references both `spell.SpellChecker` and `EditorBuffer`. It tests the integration between the two, which is editor's responsibility.

- **`ExtractWords`** is defined in spell and used only within spell's `CheckLine`. It stays in spell.
