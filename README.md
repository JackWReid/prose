# prose

A text editor for writing.

<!-- TODO: Add terminal screenshot -->

## What it does

- **Modal editing inspired by vim** -- three simple modes (Default, Edit, Line-Select) let you navigate, write, and select text without reaching for the mouse.
- **Markdown syntax highlighting** -- headers, bold, italic, code blocks, links, and lists are all colour-coded so your document is easy to scan.
- **British English spell checking** -- toggle it on and misspelled words are highlighted in real time. Acronyms and contractions are handled gracefully.
- **Distraction-free adjustable column layout** -- centre your text in the terminal and resize the column width on the fly.

## Installation

### With Go installed

```
go install github.com/JackWReid/prose/cmd/prose@latest
```

This puts the `prose` binary in your Go bin directory. To also get the man page, clone the repository and run:

```
git clone https://github.com/JackWReid/prose.git
cd prose
make install-man
```

### Build from source

```
git clone https://github.com/JackWReid/prose.git
cd prose
make install
```

This builds the binary and installs it along with the man page to `/usr/local` by default. You can change the destination with `make install PREFIX=~/.local`.

## Quick start

Open a file:

```
prose myfile.md
```

Or open several files at once (each gets its own tab):

```
prose chapter1.md chapter2.md notes.txt
```

Run `prose` with no arguments to start with an empty scratch buffer.

### The three modes

prose has three modes. If you have never used vim, think of them as three different "gears" the editor can be in.

1. **Default mode** -- This is where you start. You can move around the document, delete lines, copy and paste, search, and run commands. You cannot type text directly in this mode.
2. **Edit mode** -- This is where you type. Press `i` to enter Edit mode at the cursor, then write normally. Press `Esc` when you are done to return to Default mode.
3. **Line-Select mode** -- This is for selecting whole lines. Press `V` to start selecting, use `j` and `k` to extend the selection up or down, then copy or delete the selected lines. Press `Esc` to cancel.

A quick loop to get comfortable: press `i` to type, press `Esc` to stop typing, move around with the arrow keys (or `h` `j` `k` `l`), and press `i` again when you want to type more.

## Keybinding cheatsheet

### Default mode

#### Movement

| Key | Action |
|---|---|
| `h` `j` `k` `l` | Move left, down, up, right |
| Arrow keys | Move left, down, up, right |
| `w` | Jump to start of next word |
| `b` | Jump to start of previous word |
| `0` or `Home` | Jump to start of line |
| `$` or `End` | Jump to end of line |
| `^` | Jump to first non-whitespace character on line |
| `gg` | Jump to first line of document |
| `G` | Jump to last line of document |
| `Ctrl-U` or `Page Up` | Scroll up by one screen |
| `Ctrl-D` or `Page Down` | Scroll down by one screen |
| `Shift-Page Up` | Jump to first line (same as `gg`) |
| `Shift-Page Down` | Jump to last line (same as `G`) |
| Mouse click | Position cursor at click location |

#### Editing

| Key | Action |
|---|---|
| `dd` | Delete current line |
| `yy` | Yank (copy) current line |
| `p` | Paste below current line |
| `P` | Paste above current line |
| `u` | Undo |
| `Ctrl-R` | Redo |
| `ss` | Send current line to scratch buffer |

#### Entering Edit mode

| Key | Action |
|---|---|
| `i` | Enter Edit mode at cursor |
| `A` | Move to end of line and enter Edit mode |
| `o` | Insert new line below and enter Edit mode |
| `O` | Insert new line above and enter Edit mode |

#### Other

| Key | Action |
|---|---|
| `V` | Enter Line-Select mode |
| `S` | Jump to scratch buffer |
| `Tab` | Next tab |
| `Shift-Tab` | Previous tab |

### Edit mode

| Key | Action |
|---|---|
| Any character | Insert text at cursor |
| `Esc` | Return to Default mode |
| `Backspace` | Delete character before cursor |
| `Delete` | Delete character after cursor |
| `Enter` | Insert new line |
| Arrow keys | Move cursor |
| `Home` | Jump to start of line |
| `End` | Jump to end of line |
| Mouse click | Position cursor at click location |

### Line-Select mode

Enter with `V` from Default mode.

| Key | Action |
|---|---|
| `j` / `k` | Extend selection down / up |
| `d` | Delete selected lines |
| `y` | Yank (copy) selected lines |
| `s` | Send selected lines to scratch buffer |
| `Esc` | Cancel selection and return to Default mode |

### Leader commands (`Space` + key)

| Key | Action |
|---|---|
| `Space` then `O` | Open directory browser |
| `Space` then `H` | Open document outline (Markdown files only) |
| `Space` then `-` | Adjust column width (use left/right arrows or `h`/`l`, `Enter` to confirm, `Esc` to cancel) |

### Command mode (`:`)

Press `:` in Default mode, type a command, and press `Enter`.

| Command | Action |
|---|---|
| `:w` | Save current file |
| `:q` | Quit current tab |
| `:q!` | Quit without saving |
| `:wq` | Save and quit |
| `:qa` | Quit all tabs |
| `:qa!` | Quit all without saving |
| `:wqa` | Save all and quit all |
| `:spell` | Toggle spell checking on or off |
| `:rename newname` | Rename or move the current file |

### Search (`/`)

| Key | Action |
|---|---|
| `/` | Start a search -- type your term and press `Enter` |
| `n` | Jump to next match |
| `N` | Jump to previous match |
| `//` | Clear search highlights |
| `Esc` | Cancel search entry |

The status bar shows a match counter (e.g. "4 matches") while a search is active.

### Spell check navigation

Spell checking is off by default. Toggle it with `:spell` (works on `.md`, `.markdown`, and `.txt` files).

| Key | Action |
|---|---|
| `x` | Jump to next spelling error |
| `X` | Jump to previous spelling error |

### Directory browser (`Space-O`)

| Key | Action |
|---|---|
| `j` / `k` or arrow keys | Navigate the file list |
| `Enter` | Open file in current tab |
| `b` | Open file in a new tab |
| `Esc` | Close the browser |

### Document outline (`Space-H`)

| Key | Action |
|---|---|
| `j` / `k` or arrow keys | Navigate headers |
| `Enter` | Jump to selected header |
| `Esc` | Close the outline |

## Man page

For the full reference, run:

```
man prose
```
