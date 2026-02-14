# `prose` roadmap

## 1.2
- vim-style commands for write, quit
- `i` to enter insert/edit mode
- `rename` functionality to move the current file
- `o` for insert on line below

## 1.3
- Markdown syntax highlight colours
- Multiple file support with tabs and floating file picker

## 1.4
- `dd` for line delete from Default mode
- `A` for Insert at line end from Default mode
- `^/$` for start/end line motion from Default mode
- <Ctrl-D>/PgUp and <Ctrl-U>/PgDn for scrolling by window in Default or Insert mode
- Home/End support in Default or Insert mode
- Del in Insert mode deletes from the other side of the cursor

## 1.5
- `O` for insert on line above
- `gg`/`<Shift-PgUp>` and `G`/`<Shift-PgDn>`
- Yank/X + paste support with single yank buffer (Default mode)
- Improved undo/redo (Default mode)

## 1.6
- Mouse support for clicking and setting cursor location
- <Space>-H to open the document outline floating window (.md only)

## 1.7
- `qa`, `!qa`, and `wqa/qwa` support
- Improved floating overlay UI visuals
- <Space>-O Current directory file browsing support with floating file picker, open file with enter, open in new tab/buffer with b

## 1.8
- English language spellcheck based on some system dictionary

## 1.9
- `x` to jump cursor to next spelling error, `X` to jump to the previous
- Enter line-select mode with <Shift-V>, `y` and `d` then act on whole lines
- App-level scratch buffer. `S` to jump to the scratch buffer or create and jump if not yet created in this session.
- Send selections with `s` for lines in line-select mode and `ss` for a single line in Default mode.

## 1.10
- Slash-search. The `/` command changes the left part of the status line to the search term input. On `esc`, drop out of search. On enter, matching pieces of text are highlighted in reverse video. If no matches, drop out of search.
- `n` to jump to the next search highlight. `N` to jump to the previous search highlight.
- "4 matches" counter in the right statusline while search is active
- Double tap `//` to clear search

## 1.11
- Comprehensive user help documentation and/or man page.
- `w` to jump to next word. `b` to jump to previous word.

## 1.12
- Release tagging fixed
- Go module path fixed for proper `go install` support
- Repository maintenance and cleanup

## 1.13
- Spellcheck is off by default and is toggled with `:spell`

## 1.14
- Fix rendering flicker during rapid key presses

## 1.15
- Fix text highlights (spellcheck and search) bleeding across soft-wrapped lines
