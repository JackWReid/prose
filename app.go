package main

import (
	"os"
)

// Mode represents the editor mode.
type Mode int

const (
	ModeDefault Mode = iota
	ModeEdit
)

// App is the top-level editor state.
type App struct {
	buf       *Buffer
	undo      *UndoStack
	viewport  *Viewport
	renderer  *Renderer
	statusBar *StatusBar
	terminal  *Terminal
	mode      Mode

	// Cursor position in buffer coordinates (line, col in runes).
	cursorLine int
	cursorCol  int

	quit bool
}

func NewApp(filename string) *App {
	return &App{
		buf:       NewBuffer(filename),
		undo:      NewUndoStack(),
		renderer:  NewRenderer(),
		statusBar: NewStatusBar(),
		mode:      ModeDefault,
	}
}

func (a *App) Run() error {
	// Load file.
	if err := a.buf.Load(); err != nil {
		return err
	}

	// Set up terminal.
	t, err := NewTerminal()
	if err != nil {
		return err
	}
	a.terminal = t
	defer t.Restore()

	a.viewport = NewViewport(t.width, t.height)

	// Initial render.
	a.render()

	// Main event loop.
	for !a.quit {
		// Check for resize signal (non-blocking).
		select {
		case <-t.SigwinchChan():
			t.Resize()
			a.viewport.Resize(t.width, t.height)
			a.render()
			continue
		default:
		}

		key, err := t.ReadKey()
		if err != nil {
			return err
		}

		a.handleKey(key)
		if !a.quit {
			a.render()
		}
	}

	return nil
}

func (a *App) handleKey(key Key) {
	// If a prompt is active, handle it first.
	if a.statusBar.Prompt != PromptNone {
		a.handlePromptKey(key)
		return
	}

	switch a.mode {
	case ModeDefault:
		a.handleDefaultKey(key)
	case ModeEdit:
		a.handleEditKey(key)
	}
}

func (a *App) handleDefaultKey(key Key) {
	switch key.Type {
	case KeyRune:
		switch key.Rune {
		case 'e':
			a.mode = ModeEdit
		case 'x':
			if a.buf.Dirty {
				a.statusBar.StartPrompt(PromptQuitDirty)
			} else {
				a.quit = true
			}
		case 's':
			a.save()
		case 'r':
			a.statusBar.StartPrompt(PromptRename)
		case 'h':
			a.moveCursor(KeyLeft)
		case 'j':
			a.moveCursor(KeyDown)
		case 'k':
			a.moveCursor(KeyUp)
		case 'l':
			a.moveCursor(KeyRight)
		}
	case KeyUp, KeyDown, KeyLeft, KeyRight:
		a.moveCursor(key.Type)
	case KeyCtrlZ:
		a.undoAction()
	}
}

func (a *App) handleEditKey(key Key) {
	switch key.Type {
	case KeyEscape:
		a.mode = ModeDefault
	case KeyRune:
		a.insertChar(key.Rune)
	case KeyEnter:
		a.insertNewline()
	case KeyBackspace:
		a.deleteChar()
	case KeyUp, KeyDown, KeyLeft, KeyRight:
		a.moveCursor(key.Type)
	case KeyCtrlZ:
		a.undoAction()
	}
}

func (a *App) handlePromptKey(key Key) {
	switch a.statusBar.Prompt {
	case PromptQuitDirty:
		switch key.Type {
		case KeyRune:
			switch key.Rune {
			case 'x':
				a.quit = true
			case 's':
				a.statusBar.ClearPrompt()
				a.save()
			}
		case KeyEscape:
			a.statusBar.ClearPrompt()
		}

	case PromptRename, PromptSaveNew:
		text, done, cancelled := a.statusBar.HandlePromptKey(key)
		if cancelled {
			return
		}
		if done && text != "" {
			a.buf.Save(text)
		}
	}
}

func (a *App) save() {
	if a.buf.Filename == "" {
		a.statusBar.StartPrompt(PromptSaveNew)
		return
	}
	a.buf.Save("")
}

// insertChar inserts a character at the cursor and advances the cursor.
func (a *App) insertChar(ch rune) {
	a.buf.InsertChar(a.cursorLine, a.cursorCol, ch)
	a.undo.PushInsertChar(a.cursorLine, a.cursorCol, ch)
	a.cursorCol++
}

// insertNewline splits the current line at the cursor.
func (a *App) insertNewline() {
	a.undo.PushInsertLine(a.cursorLine, a.cursorCol, a.cursorLine, a.cursorCol)
	a.buf.InsertNewline(a.cursorLine, a.cursorCol)
	a.cursorLine++
	a.cursorCol = 0
}

// deleteChar deletes the character before the cursor (backspace).
func (a *App) deleteChar() {
	if a.cursorCol == 0 && a.cursorLine == 0 {
		return
	}

	if a.cursorCol > 0 {
		// Delete character within the line.
		ch, _ := a.buf.DeleteChar(a.cursorLine, a.cursorCol)
		if ch == 0 {
			return
		}
		a.undo.PushDeleteChar(a.cursorLine, a.cursorCol-1, ch, a.cursorLine, a.cursorCol)
		a.cursorCol--
	} else {
		// At column 0: join with the previous line.
		// Capture the previous line's length before the join — that's where the cursor goes.
		prevLineLen := a.buf.LineLen(a.cursorLine - 1)
		saveLine := a.cursorLine
		saveCol := a.cursorCol

		a.buf.JoinLines(a.cursorLine - 1)
		a.buf.Dirty = true
		a.undo.PushDeleteLine(a.cursorLine-1, prevLineLen, saveLine, saveCol)

		a.cursorLine--
		a.cursorCol = prevLineLen
	}
}

// moveCursor moves the cursor in the given direction, clamping to valid positions.
func (a *App) moveCursor(dir int) {
	switch dir {
	case KeyLeft:
		if a.cursorCol > 0 {
			a.cursorCol--
		} else if a.cursorLine > 0 {
			a.cursorLine--
			a.cursorCol = a.buf.LineLen(a.cursorLine)
		}
	case KeyRight:
		if a.cursorCol < a.buf.LineLen(a.cursorLine) {
			a.cursorCol++
		} else if a.cursorLine < a.buf.LineCount()-1 {
			a.cursorLine++
			a.cursorCol = 0
		}
	case KeyUp:
		if a.cursorLine > 0 {
			a.cursorLine--
			if a.cursorCol > a.buf.LineLen(a.cursorLine) {
				a.cursorCol = a.buf.LineLen(a.cursorLine)
			}
		}
	case KeyDown:
		if a.cursorLine < a.buf.LineCount()-1 {
			a.cursorLine++
			if a.cursorCol > a.buf.LineLen(a.cursorLine) {
				a.cursorCol = a.buf.LineLen(a.cursorLine)
			}
		}
	}
}

func (a *App) undoAction() {
	line, col, ok := a.undo.Undo(a.buf)
	if ok {
		a.cursorLine = line
		a.cursorCol = col
	}
}

func (a *App) render() {
	displayLines := WrapBuffer(a.buf, a.viewport.ColWidth)
	cursorDL, cursorDC := CursorToDisplayLine(displayLines, a.cursorLine, a.cursorCol)
	a.viewport.EnsureCursorVisible(cursorDL)

	statusLeft := a.statusBar.FormatLeft(a.buf.Filename, a.buf.Dirty)
	statusRight := a.statusBar.FormatRight(a.mode)

	frame := a.renderer.RenderFrame(displayLines, a.viewport, cursorDL, cursorDC, statusLeft, statusRight)
	os.Stdout.WriteString(frame)
}
