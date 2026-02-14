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
	outline   *Outline
	mode      Mode

	leaderPending bool   // Space was pressed, awaiting second key.
	dPending      bool   // 'd' was pressed, awaiting second 'd' for dd.
	gPending      bool   // 'g' was pressed, awaiting second 'g' for gg.
	yPending      bool   // 'y' was pressed, awaiting second 'y' for yy.
	yankBuffer    string // Shared yank buffer for yy/dd/p/P operations.
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
		outline:   &Outline{},
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

		event, err := t.ReadKey()
		if err != nil {
			return err
		}

		a.handleInput(event)
		if !a.quit {
			a.render()
		}
	}

	return nil
}

func (a *App) handleInput(event InputEvent) {
	// Clear any temporary status message on input.
	a.statusBar.ClearMessage()

	// Handle mouse events.
	if event.Type == EventMouse {
		a.handleMouse(event.Mouse)
		return
	}

	// Handle keyboard events.
	key := event.Key

	// If outline is active, handle it first.
	if a.outline.Active {
		a.handleOutlineKey(key)
		return
	}

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

func (a *App) handleMouse(mouse MouseEvent) {
	// Ignore mouse events when outline, picker, or prompt is active.
	if a.outline.Active || a.picker.Active || a.statusBar.Prompt != PromptNone {
		return
	}

	// Only handle left button press for now.
	if mouse.Button != MouseLeft || !mouse.Press {
		return
	}

	// Convert mouse coordinates to buffer position.
	line, col := a.mouseToBufferPos(mouse.Row, mouse.Col)
	if line >= 0 && col >= 0 {
		eb := a.currentBuf()
		eb.cursorLine = line
		eb.cursorCol = col
	}
}

func (a *App) handleDefaultKey(key Key) {
	// Leader key sequence: Space followed by a second key.
	if a.leaderPending {
		a.leaderPending = false
		if key.Type == KeyRune {
			switch key.Rune {
			case 'b':
				a.picker.Show(a.currentBuffer)
				return
			case 'h', 'H':
				a.showOutline()
				return
			}
		}
		// Unknown leader combo — ignore.
		return
	}

	// dd operator: 'd' followed by 'd'.
	if a.dPending {
		a.dPending = false
		if key.Type == KeyRune && key.Rune == 'd' {
			a.deleteWholeLine()
			return
		}
		// Not 'dd' — consume the key and cancel.
		return
	}

	// gg operator: 'g' followed by 'g'.
	if a.gPending {
		a.gPending = false
		if key.Type == KeyRune && key.Rune == 'g' {
			a.jumpToTop()
			return
		}
		// Not 'gg' — consume the key and cancel.
		return
	}

	// yy operator: 'y' followed by 'y'.
	if a.yPending {
		a.yPending = false
		if key.Type == KeyRune && key.Rune == 'y' {
			a.yankLine()
			return
		}
		// Not 'yy' — consume the key and cancel.
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
		case 'O':
			eb.buf.InsertLine(eb.cursorLine, "")
			eb.undo.PushInsertWholeLine(eb.cursorLine)
			eb.cursorCol = 0
			a.mode = ModeEdit
		case 'd':
			a.dPending = true
		case 'y':
			a.yPending = true
		case 'p':
			a.pasteBelow()
		case 'P':
			a.pasteAbove()
		case 'u':
			a.undoAction()
		case 'g':
			a.gPending = true
		case 'G':
			a.jumpToBottom()
		case 'A':
			eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
			a.mode = ModeEdit
		case '^':
			// Jump to first non-whitespace character.
			runes := []rune(eb.buf.Lines[eb.cursorLine])
			for i, r := range runes {
				if r != ' ' && r != '\t' {
					eb.cursorCol = i
					return
				}
			}
			eb.cursorCol = 0
		case '$':
			eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
		}
	case KeyUp, KeyDown, KeyLeft, KeyRight:
		a.moveCursor(key.Type)
	case KeyHome:
		eb.cursorCol = 0
	case KeyEnd:
		eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
	case KeyCtrlD:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollDown(visibleLines / 2)
	case KeyCtrlU:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollUp(visibleLines / 2)
	case KeyPgDn:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollDown(visibleLines)
	case KeyPgUp:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollUp(visibleLines)
	case KeyCtrlZ:
		a.undoAction()
	case KeyCtrlY:
		a.redoAction()
	case KeyCtrlR:
		a.redoAction()
	}
}

func (a *App) handleEditKey(key Key) {
	// Clear any pending operators from Default mode.
	a.dPending = false
	a.gPending = false
	a.yPending = false

	eb := a.currentBuf()
	switch key.Type {
	case KeyEscape:
		a.mode = ModeDefault
	case KeyRune:
		a.insertChar(key.Rune)
	case KeyEnter:
		a.insertNewline()
	case KeyBackspace:
		a.deleteChar()
	case KeyDelete:
		a.deleteCharForward()
	case KeyUp, KeyDown, KeyLeft, KeyRight:
		a.moveCursor(key.Type)
	case KeyHome:
		eb.cursorCol = 0
	case KeyEnd:
		eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
	case KeyCtrlD:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollDown(visibleLines / 2)
	case KeyCtrlU:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollUp(visibleLines / 2)
	case KeyPgDn:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollDown(visibleLines)
	case KeyPgUp:
		visibleLines := a.viewport.VisibleLines(eb.scrollOffset)
		a.scrollUp(visibleLines)
	case KeyCtrlZ:
		a.undoAction()
	case KeyCtrlY:
		a.redoAction()
	case KeyCtrlR:
		a.redoAction()
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

func (a *App) handleOutlineKey(key Key) {
	switch key.Type {
	case KeyEscape:
		a.outline.Hide()
	case KeyUp:
		a.outline.MoveUp()
	case KeyDown:
		a.outline.MoveDown()
	case KeyRune:
		switch key.Rune {
		case 'k':
			a.outline.MoveUp()
		case 'j':
			a.outline.MoveDown()
		}
	case KeyEnter:
		a.jumpToOutlineItem()
		a.outline.Hide()
	}
}

func (a *App) showOutline() {
	eb := a.currentBuf()

	// Check if file is markdown.
	if !IsMarkdownFile(eb.buf.Filename) {
		a.statusBar.SetMessage("Outline only available for markdown files")
		return
	}

	// Extract headings.
	items := ExtractHeadings(eb.buf)
	if len(items) == 0 {
		a.statusBar.SetMessage("No headings found")
		return
	}

	a.outline.Show(items)
}

func (a *App) jumpToOutlineItem() {
	if a.outline.Selected < 0 || a.outline.Selected >= len(a.outline.Items) {
		return
	}

	item := a.outline.Items[a.outline.Selected]
	eb := a.currentBuf()
	eb.cursorLine = item.BufferLine
	eb.cursorCol = 0
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

func (a *App) jumpToTop() {
	eb := a.currentBuf()
	eb.cursorLine = 0
	eb.cursorCol = 0
}

func (a *App) jumpToBottom() {
	eb := a.currentBuf()
	eb.cursorLine = eb.buf.LineCount() - 1
	eb.cursorCol = 0
}

func (a *App) yankLine() {
	eb := a.currentBuf()
	a.yankBuffer = eb.buf.Lines[eb.cursorLine]
	a.statusBar.SetMessage("Yanked line")
}

func (a *App) pasteBelow() {
	if a.yankBuffer == "" {
		return
	}
	eb := a.currentBuf()
	eb.buf.InsertLine(eb.cursorLine+1, a.yankBuffer)
	eb.undo.PushInsertWholeLine(eb.cursorLine + 1)
	eb.cursorLine++
	eb.cursorCol = 0
}

func (a *App) pasteAbove() {
	if a.yankBuffer == "" {
		return
	}
	eb := a.currentBuf()
	eb.buf.InsertLine(eb.cursorLine, a.yankBuffer)
	eb.undo.PushInsertWholeLine(eb.cursorLine)
	eb.cursorCol = 0
}

func (a *App) undoAction() {
	eb := a.currentBuf()
	line, col, ok := eb.undo.Undo(eb.buf)
	if ok {
		eb.cursorLine = line
		eb.cursorCol = col
	}
}

func (a *App) redoAction() {
	eb := a.currentBuf()
	line, col, ok := eb.undo.Redo(eb.buf)
	if ok {
		eb.cursorLine = line
		eb.cursorCol = col
	}
}

// deleteWholeLine deletes the entire current line (dd operation).
func (a *App) deleteWholeLine() {
	eb := a.currentBuf()
	content := eb.buf.DeleteLine(eb.cursorLine)
	a.yankBuffer = content // Populate yank buffer for cut semantics.
	eb.undo.PushDeleteWholeLine(eb.cursorLine, content, eb.cursorLine, eb.cursorCol)

	// Clamp cursor position after deletion.
	if eb.cursorLine >= eb.buf.LineCount() {
		eb.cursorLine = eb.buf.LineCount() - 1
	}
	if eb.cursorCol > eb.buf.LineLen(eb.cursorLine) {
		eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
	}
}

// deleteCharForward deletes the character at the cursor position (Del key).
func (a *App) deleteCharForward() {
	eb := a.currentBuf()
	lineLen := eb.buf.LineLen(eb.cursorLine)

	if eb.cursorCol < lineLen {
		// Delete character at cursor position.
		ch := eb.buf.DeleteCharForward(eb.cursorLine, eb.cursorCol)
		if ch != 0 {
			eb.undo.PushDeleteChar(eb.cursorLine, eb.cursorCol, ch, eb.cursorLine, eb.cursorCol)
		}
	} else if eb.cursorLine < eb.buf.LineCount()-1 {
		// At end of line: join with next line.
		eb.buf.JoinLines(eb.cursorLine)
		eb.undo.PushDeleteLine(eb.cursorLine, lineLen, eb.cursorLine, eb.cursorCol)
	}
}

// scrollDown moves the cursor down by n lines.
func (a *App) scrollDown(n int) {
	eb := a.currentBuf()
	eb.cursorLine += n
	if eb.cursorLine >= eb.buf.LineCount() {
		eb.cursorLine = eb.buf.LineCount() - 1
	}
	// Clamp column to new line length.
	if eb.cursorCol > eb.buf.LineLen(eb.cursorLine) {
		eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
	}
}

// scrollUp moves the cursor up by n lines.
func (a *App) scrollUp(n int) {
	eb := a.currentBuf()
	eb.cursorLine -= n
	if eb.cursorLine < 0 {
		eb.cursorLine = 0
	}
	// Clamp column to new line length.
	if eb.cursorCol > eb.buf.LineLen(eb.cursorLine) {
		eb.cursorCol = eb.buf.LineLen(eb.cursorLine)
	}
}

// mouseToBufferPos converts terminal mouse coordinates to buffer line/col.
// Returns (-1, -1) if the click is outside the text area.
func (a *App) mouseToBufferPos(termRow, termCol int) (int, int) {
	eb := a.currentBuf()
	vp := a.viewport

	// Account for top padding (1 line when scrollOffset == 0).
	topPadding := 0
	if eb.scrollOffset == 0 {
		topPadding = 1
	}

	// Click on status bar or above text area — ignore.
	if termRow == vp.Height || termRow < 1+topPadding {
		return -1, -1
	}

	// Convert terminal row to display line index.
	displayLineIdx := eb.scrollOffset + (termRow - 1 - topPadding)

	// Generate wrapped display lines.
	displayLines := WrapBuffer(eb.buf, vp.ColWidth)

	// Check if click is beyond the last display line.
	if displayLineIdx >= len(displayLines) {
		// Click below text — place cursor at end of last line.
		if len(displayLines) > 0 {
			lastDL := displayLines[len(displayLines)-1]
			line := lastDL.BufferLine
			col := eb.buf.LineLen(line)
			return line, col
		}
		return -1, -1
	}

	dl := displayLines[displayLineIdx]
	bufferLine := dl.BufferLine

	// Account for left margin in column calculation.
	clickCol := termCol - 1 - vp.LeftMargin
	if clickCol < 0 {
		clickCol = 0
	}

	// Map display column to buffer column.
	// The display line shows text starting at dl.Offset in the buffer line.
	bufferCol := dl.Offset + clickCol

	// Clamp to actual line length.
	lineLen := eb.buf.LineLen(bufferLine)
	if bufferCol > lineLen {
		bufferCol = lineLen
	}

	return bufferLine, bufferCol
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

	// Render outline overlay if active.
	if a.outline.Active {
		frame += a.renderer.RenderOutline(a.outline, a.viewport)
	}

	os.Stdout.WriteString(frame)
}

func formatBufferInfo(current, total int) string {
	return fmt.Sprintf("[%d/%d]", current, total)
}
