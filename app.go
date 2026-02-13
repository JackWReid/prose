package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mode represents the editor mode.
type Mode int

const (
	ModeDefault Mode = iota
	ModeEdit
)

// App is the top-level editor state.
type App struct {
	buffers       []*EditorBuffer
	currentBuffer int

	viewport  *Viewport
	renderer  *Renderer
	statusBar *StatusBar
	terminal  *Terminal
	picker    *Picker
	mode      Mode

	leaderPending bool  // Space was pressed, awaiting second key.
	quit          bool
	quitAfterSave bool // Set by :wq on unnamed buffers.
}

// currentBuf returns the active EditorBuffer.
func (a *App) currentBuf() *EditorBuffer {
	return a.buffers[a.currentBuffer]
}

func NewApp(filenames []string) *App {
	app := &App{
		renderer:  NewRenderer(),
		statusBar: NewStatusBar(),
		picker:    &Picker{},
		mode:      ModeDefault,
	}
	if len(filenames) == 0 {
		app.buffers = []*EditorBuffer{NewEditorBuffer("")}
	} else {
		for _, f := range filenames {
			app.buffers = append(app.buffers, NewEditorBuffer(f))
		}
	}
	return app
}

func (a *App) Run() error {
	// Load all buffers.
	for _, eb := range a.buffers {
		if err := eb.buf.Load(); err != nil {
			return err
		}
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
	// Clear any temporary status message on keypress.
	a.statusBar.ClearMessage()

	// If picker is active, handle it first.
	if a.picker.Active {
		a.handlePickerKey(key)
		return
	}

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
	// Leader key sequence: Space followed by a second key.
	if a.leaderPending {
		a.leaderPending = false
		if key.Type == KeyRune {
			switch key.Rune {
			case 'p':
				a.picker.Show(a.currentBuffer)
				return
			}
		}
		// Unknown leader combo — ignore.
		return
	}

	eb := a.currentBuf()
	switch key.Type {
	case KeyRune:
		switch key.Rune {
		case ' ':
			a.leaderPending = true
		case 'i':
			a.mode = ModeEdit
		case ':':
			a.statusBar.StartPrompt(PromptCommand)
		case 'h':
			a.moveCursor(KeyLeft)
		case 'j':
			a.moveCursor(KeyDown)
		case 'k':
			a.moveCursor(KeyUp)
		case 'l':
			a.moveCursor(KeyRight)
		case 'o':
			eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
			a.insertNewline()
			a.mode = ModeEdit
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

func (a *App) handlePickerKey(key Key) {
	switch key.Type {
	case KeyEscape:
		a.picker.Hide()
	case KeyUp:
		a.picker.MoveUp()
	case KeyDown:
		a.picker.MoveDown(len(a.buffers))
	case KeyRune:
		switch key.Rune {
		case 'k':
			a.picker.MoveUp()
		case 'j':
			a.picker.MoveDown(len(a.buffers))
		}
	case KeyEnter:
		a.currentBuffer = a.picker.Selected
		a.picker.Hide()
	}
}

func (a *App) handlePromptKey(key Key) {
	eb := a.currentBuf()
	switch a.statusBar.Prompt {
	case PromptSaveNew:
		text, done, cancelled := a.statusBar.HandlePromptKey(key)
		if cancelled {
			a.quitAfterSave = false
			return
		}
		if done && text != "" {
			eb.buf.Save(text)
			eb.highlighter = DetectHighlighter(eb.buf.Filename)
			if a.quitAfterSave {
				a.closeCurrentBuffer()
				a.quitAfterSave = false
			}
		}

	case PromptCommand:
		text, done, cancelled := a.statusBar.HandlePromptKey(key)
		if cancelled {
			return
		}
		if done {
			a.executeCommand(text)
		}
	}
}

func (a *App) executeCommand(cmd string) {
	eb := a.currentBuf()
	cmd = strings.TrimSpace(cmd)

	switch {
	case cmd == "q":
		if eb.buf.Dirty {
			a.statusBar.SetMessage("Unsaved changes. Use :q! to discard, or :w to save.")
		} else {
			a.closeCurrentBuffer()
		}

	case cmd == "q!":
		a.closeCurrentBuffer()

	case cmd == "w":
		a.save()

	case strings.HasPrefix(cmd, "w "):
		filename := strings.TrimSpace(cmd[2:])
		if filename != "" {
			eb.buf.Save(filename)
			eb.highlighter = DetectHighlighter(eb.buf.Filename)
		}

	case cmd == "wq":
		if eb.buf.Filename == "" {
			a.quitAfterSave = true
			a.statusBar.StartPrompt(PromptSaveNew)
		} else {
			eb.buf.Save("")
			a.closeCurrentBuffer()
		}

	case strings.HasPrefix(cmd, "e "):
		filename := strings.TrimSpace(cmd[2:])
		if filename == "" {
			a.statusBar.SetMessage("Usage: :e <filename>")
			return
		}
		idx := a.openBuffer(filename)
		a.currentBuffer = idx

	case cmd == "e":
		a.statusBar.SetMessage("Usage: :e <filename>")

	case strings.HasPrefix(cmd, "rename "):
		newName := strings.TrimSpace(cmd[7:])
		if newName == "" {
			return
		}
		oldName := eb.buf.Filename
		if oldName == "" {
			// Unnamed buffer — behaves like :w <filename>.
			eb.buf.Save(newName)
			eb.highlighter = DetectHighlighter(eb.buf.Filename)
		} else {
			if err := os.Rename(oldName, newName); err != nil {
				a.statusBar.SetMessage("Rename failed: " + err.Error())
				return
			}
			eb.buf.Filename = newName
			eb.highlighter = DetectHighlighter(eb.buf.Filename)
		}

	default:
		a.statusBar.SetMessage("Unknown command: " + cmd)
	}
}

// openBuffer opens a file or switches to it if already open. Returns the buffer index.
func (a *App) openBuffer(filename string) int {
	// Normalise to absolute path for comparison.
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename
	}

	// Check if already open.
	for i, eb := range a.buffers {
		existingPath, err2 := filepath.Abs(eb.buf.Filename)
		if err2 != nil {
			existingPath = eb.buf.Filename
		}
		if existingPath == absPath {
			return i
		}
	}

	// Create new buffer.
	eb := NewEditorBuffer(filename)
	eb.buf.Load()
	a.buffers = append(a.buffers, eb)
	return len(a.buffers) - 1
}

// closeCurrentBuffer removes the current buffer. If it's the last one, quit.
func (a *App) closeCurrentBuffer() {
	if len(a.buffers) == 1 {
		a.quit = true
		return
	}
	a.buffers = append(a.buffers[:a.currentBuffer], a.buffers[a.currentBuffer+1:]...)
	if a.currentBuffer >= len(a.buffers) {
		a.currentBuffer = len(a.buffers) - 1
	}
}

func (a *App) save() {
	eb := a.currentBuf()
	if eb.buf.Filename == "" {
		a.statusBar.StartPrompt(PromptSaveNew)
		return
	}
	eb.buf.Save("")
}

// insertChar inserts a character at the cursor and advances the cursor.
func (a *App) insertChar(ch rune) {
	eb := a.currentBuf()
	eb.buf.InsertChar(eb.cursorLine, eb.cursorCol, ch)
	eb.undo.PushInsertChar(eb.cursorLine, eb.cursorCol, ch)
	eb.cursorCol++
}

// insertNewline splits the current line at the cursor.
func (a *App) insertNewline() {
	eb := a.currentBuf()
	eb.undo.PushInsertLine(eb.cursorLine, eb.cursorCol, eb.cursorLine, eb.cursorCol)
	eb.buf.InsertNewline(eb.cursorLine, eb.cursorCol)
	eb.cursorLine++
	eb.cursorCol = 0
}

// deleteChar deletes the character before the cursor (backspace).
func (a *App) deleteChar() {
	eb := a.currentBuf()
	if eb.cursorCol == 0 && eb.cursorLine == 0 {
		return
	}

	if eb.cursorCol > 0 {
		// Delete character within the line.
		ch, _ := eb.buf.DeleteChar(eb.cursorLine, eb.cursorCol)
		if ch == 0 {
			return
		}
		eb.undo.PushDeleteChar(eb.cursorLine, eb.cursorCol-1, ch, eb.cursorLine, eb.cursorCol)
		eb.cursorCol--
	} else {
		// At column 0: join with the previous line.
		prevLineLen := eb.buf.LineLen(eb.cursorLine - 1)
		saveLine := eb.cursorLine
		saveCol := eb.cursorCol

		eb.buf.JoinLines(eb.cursorLine - 1)
		eb.buf.Dirty = true
		eb.undo.PushDeleteLine(eb.cursorLine-1, prevLineLen, saveLine, saveCol)

		eb.cursorLine--
		eb.cursorCol = prevLineLen
	}
}

// moveCursor moves the cursor in the given direction, clamping to valid positions.
func (a *App) moveCursor(dir int) {
	eb := a.currentBuf()
	switch dir {
	case KeyLeft:
		if eb.cursorCol > 0 {
			eb.cursorCol--
		} else if eb.cursorLine > 0 {
			eb.cursorLine--
			eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
		}
	case KeyRight:
		if eb.cursorCol < eb.buf.LineLen(eb.cursorLine) {
			eb.cursorCol++
		} else if eb.cursorLine < eb.buf.LineCount()-1 {
			eb.cursorLine++
			eb.cursorCol = 0
		}
	case KeyUp:
		if eb.cursorLine > 0 {
			eb.cursorLine--
			if eb.cursorCol > eb.buf.LineLen(eb.cursorLine) {
				eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
			}
		}
	case KeyDown:
		if eb.cursorLine < eb.buf.LineCount()-1 {
			eb.cursorLine++
			if eb.cursorCol > eb.buf.LineLen(eb.cursorLine) {
				eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
			}
		}
	}
}

func (a *App) undoAction() {
	eb := a.currentBuf()
	line, col, ok := eb.undo.Undo(eb.buf)
	if ok {
		eb.cursorLine = line
		eb.cursorCol = col
	}
}

func (a *App) render() {
	eb := a.currentBuf()
	displayLines := WrapBuffer(eb.buf, a.viewport.ColWidth)
	cursorDL, cursorDC := CursorToDisplayLine(displayLines, eb.cursorLine, eb.cursorCol)

	a.viewport.EnsureCursorVisible(cursorDL, &eb.scrollOffset)

	bufferInfo := ""
	if len(a.buffers) > 1 {
		bufferInfo = formatBufferInfo(a.currentBuffer+1, len(a.buffers))
	}

	statusLeft := a.statusBar.FormatLeft(eb.Filename(), eb.IsDirty(), bufferInfo)
	statusRight := a.statusBar.FormatRight(a.mode, eb.WordCount())

	frame := a.renderer.RenderFrame(displayLines, a.viewport, eb.scrollOffset, cursorDL, cursorDC, statusLeft, statusRight, eb.highlighter)

	// Render picker overlay if active.
	if a.picker.Active {
		frame += a.renderer.RenderPicker(a.buffers, a.picker, a.currentBuffer, a.viewport)
	}

	os.Stdout.WriteString(frame)
}

func formatBufferInfo(current, total int) string {
	return fmt.Sprintf("[%d/%d]", current, total)
}
