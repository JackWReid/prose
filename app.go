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
	ModeLineSelect
)

// App is the top-level editor state.
type App struct {
	buffers       []*EditorBuffer
	currentBuffer int

	viewport         *Viewport
	renderer         *Renderer
	statusBar        *StatusBar
	terminal         *Terminal
	picker           *Picker
	outline          *Outline
	browser          *Browser
	columnAdjust     *ColumnAdjust
	spellChecker     *SpellChecker
	spellCheckEnabled bool // Global toggle for spell checking (default: false).
	mode             Mode

	leaderPending    bool   // Space was pressed, awaiting second key.
	dPending         bool   // 'd' was pressed, awaiting second 'd' for dd.
	gPending         bool   // 'g' was pressed, awaiting second 'g' for gg.
	yPending         bool   // 'y' was pressed, awaiting second 'y' for yy.
	sPending         bool   // 's' was pressed, awaiting second 's' for ss.
	lineSelectAnchor int    // Line where Shift-V was pressed (for line-select mode).
	yankBuffer       string // Shared yank buffer for yy/dd/p/P operations.
	quit             bool
	quitAfterSave    bool // Set by :wq on unnamed buffers.
}

// currentBuf returns the active EditorBuffer.
func (a *App) currentBuf() *EditorBuffer {
	return a.buffers[a.currentBuffer]
}

func NewApp(filenames []string) *App {
	app := &App{
		renderer:          NewRenderer(),
		statusBar:         NewStatusBar(),
		picker:            &Picker{},
		outline:           &Outline{},
		browser:           &Browser{},
		columnAdjust:      &ColumnAdjust{},
		mode:              ModeDefault,
		spellCheckEnabled: false, // Spellcheck is off by default.
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

	// Initialize spell checker.
	spellChecker, err := NewSpellChecker()
	if err != nil {
		return fmt.Errorf("failed to initialize spell checker: %v", err)
	}
	a.spellChecker = spellChecker

	// Run initial spell check on all buffers that should be checked (if enabled).
	for _, eb := range a.buffers {
		if a.spellCheckEnabled && eb.ShouldSpellCheck() {
			eb.spellErrors = nil
			for i := 0; i < len(eb.buf.Lines); i++ {
				lineErrors := spellChecker.CheckLine(i, eb.buf.Lines[i])
				eb.spellErrors = append(eb.spellErrors, lineErrors...)
			}
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
		// Perform debounced spell checking (if enabled).
		if a.spellCheckEnabled {
			a.currentBuf().PerformSpellCheck(a.spellChecker)
		}

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

	// If column adjuster is active, handle it first.
	if a.columnAdjust.Active {
		a.handleColumnAdjustKey(key)
		return
	}

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

	// If browser is active, handle it first.
	if a.browser.Active {
		a.handleBrowserKey(key)
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
	case ModeLineSelect:
		a.handleLineSelectKey(key)
	}
}

func (a *App) handleMouse(mouse MouseEvent) {
	// Ignore mouse events when overlay or prompt is active.
	if a.columnAdjust.Active || a.outline.Active || a.picker.Active || a.browser.Active || a.statusBar.Prompt != PromptNone {
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
	// ss operator: 's' followed by 's'.
	if a.sPending {
		a.sPending = false
		if key.Type == KeyRune && key.Rune == 's' {
			a.sendCurrentLineToScratch()
			return
		}
		// Not 'ss' — cancel.
		return
	}

	// Leader key sequence: Space followed by a second key.
	if a.leaderPending {
		a.leaderPending = false
		if key.Type == KeyRune {
			switch key.Rune {
			case 'b', 't':
				a.picker.Show(a.currentBuffer)
				return
			case 'h', 'H':
				a.showOutline()
				return
			case 'o', 'O':
				a.showBrowser()
				return
			case '-':
				a.showColumnAdjust()
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
		case '/':
			a.statusBar.StartPrompt(PromptSearch)
		case 'n':
			// Jump to next search match if search is active
			if eb.searchActive {
				a.jumpToNextMatch()
			}
		case 'N':
			// Jump to previous search match if search is active
			if eb.searchActive {
				a.jumpToPrevMatch()
			}
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
		case 's':
			a.sPending = true
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
		case 'x':
			a.jumpToNextSpellError()
		case 'X':
			a.jumpToPrevSpellError()
		case 'w':
			a.jumpToNextWord()
		case 'b':
			a.jumpToPrevWord()
		case 'S':
			a.jumpToScratch()
		case 'V':
			a.mode = ModeLineSelect
			a.lineSelectAnchor = eb.cursorLine
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
	a.sPending = false

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

func (a *App) handleLineSelectKey(key Key) {
	eb := a.currentBuf()
	switch key.Type {
	case KeyEscape:
		a.mode = ModeDefault
	case KeyRune:
		switch key.Rune {
		case 'h':
			a.moveCursor(KeyLeft)
		case 'j':
			a.moveCursor(KeyDown)
		case 'k':
			a.moveCursor(KeyUp)
		case 'l':
			a.moveCursor(KeyRight)
		case 'y':
			a.yankSelectedLines()
			a.mode = ModeDefault
		case 'd':
			a.deleteSelectedLines()
			a.mode = ModeDefault
		case 's':
			a.sendSelectedLinesToScratch()
			a.mode = ModeDefault
		case 'g':
			a.gPending = true
		case 'G':
			a.jumpToBottom()
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
	}

	// Handle gg operator
	if a.gPending {
		a.gPending = false
		if key.Type == KeyRune && key.Rune == 'g' {
			a.jumpToTop()
		}
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

func (a *App) showBrowser() {
	eb := a.currentBuf()

	// Determine directory to browse.
	dir := "."
	if eb.buf.Filename != "" {
		dir = filepath.Dir(eb.buf.Filename)
	}

	// Show browser.
	if err := a.browser.Show(dir); err != nil {
		a.statusBar.SetMessage("Error opening directory: " + err.Error())
		return
	}

	// Show message if directory is empty.
	if len(a.browser.Items) == 0 {
		a.statusBar.SetMessage("Directory is empty")
		a.browser.Hide()
	}
}

func (a *App) showColumnAdjust() {
	a.columnAdjust.Show(a.viewport.TargetColWidth)
}

func (a *App) handleColumnAdjustKey(key Key) {
	switch key.Type {
	case KeyEscape:
		// Cancel — restore original width.
		a.viewport.TargetColWidth = a.columnAdjust.OrigWidth
		a.viewport.recalcLayout()
		a.columnAdjust.Hide()
	case KeyEnter:
		// Confirm — keep current width.
		a.columnAdjust.Hide()
	case KeyLeft:
		a.columnAdjust.Decrease()
		a.viewport.TargetColWidth = a.columnAdjust.Width
		a.viewport.recalcLayout()
	case KeyRight:
		a.columnAdjust.Increase(a.viewport.Width)
		a.viewport.TargetColWidth = a.columnAdjust.Width
		a.viewport.recalcLayout()
	case KeyRune:
		switch key.Rune {
		case 'h':
			a.columnAdjust.Decrease()
			a.viewport.TargetColWidth = a.columnAdjust.Width
			a.viewport.recalcLayout()
		case 'l':
			a.columnAdjust.Increase(a.viewport.Width)
			a.viewport.TargetColWidth = a.columnAdjust.Width
			a.viewport.recalcLayout()
		}
	}
}

func (a *App) handleBrowserKey(key Key) {
	switch key.Type {
	case KeyEscape:
		a.browser.Hide()
	case KeyUp:
		a.browser.MoveUp()
	case KeyDown:
		a.browser.MoveDown()
	case KeyLeft:
		a.navigateToParentDirectory()
	case KeyRune:
		switch key.Rune {
		case 'k':
			a.browser.MoveUp()
		case 'j':
			a.browser.MoveDown()
		case 'h':
			a.navigateToParentDirectory()
		case 'b', 't':
			// Open in new buffer.
			a.openBrowserItemNewBuffer()
			a.browser.Hide()
		}
	case KeyEnter:
		a.openBrowserItem()
	}
}

func (a *App) navigateToParentDirectory() {
	if a.browser.CurrentDir == "" {
		return
	}

	// Get parent directory.
	parentDir := filepath.Dir(a.browser.CurrentDir)

	// Don't navigate above root.
	if parentDir == a.browser.CurrentDir {
		return
	}

	// Navigate to parent.
	if err := a.browser.Show(parentDir); err != nil {
		a.statusBar.SetMessage("Error opening parent directory: " + err.Error())
		a.browser.Hide()
	}
}

func (a *App) openBrowserItem() {
	item := a.browser.SelectedItem()
	if item == nil {
		return
	}

	if item.IsDir {
		// Navigate into subdirectory.
		if err := a.browser.Show(item.Path); err != nil {
			a.statusBar.SetMessage("Error opening directory: " + err.Error())
			a.browser.Hide()
		} else if len(a.browser.Items) == 0 {
			a.statusBar.SetMessage("Directory is empty")
			a.browser.Hide()
		}
	} else {
		// Open file in current buffer.
		idx := a.openBuffer(item.Path)
		a.currentBuffer = idx
		a.browser.Hide()
	}
}

func (a *App) openBrowserItemNewBuffer() {
	item := a.browser.SelectedItem()
	if item == nil {
		return
	}

	if item.IsDir {
		a.statusBar.SetMessage("Cannot open directory in buffer")
		return
	}

	// Force new buffer by opening the file.
	idx := a.openBuffer(item.Path)
	a.currentBuffer = idx
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

	case PromptSearch:
		text, done, cancelled := a.statusBar.HandlePromptKey(key)
		if cancelled {
			// Clear search on escape
			a.clearSearch()
			return
		}
		if done {
			if text != "" {
				a.activateSearch(text)
			}
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
		if eb.isScratch {
			a.statusBar.SetMessage("Cannot save scratch buffer")
		} else {
			a.save()
		}

	case strings.HasPrefix(cmd, "w "):
		if eb.isScratch {
			a.statusBar.SetMessage("Cannot save scratch buffer")
		} else {
			filename := strings.TrimSpace(cmd[2:])
			if filename != "" {
				eb.buf.Save(filename)
				eb.highlighter = DetectHighlighter(eb.buf.Filename)
			}
		}

	case cmd == "wq":
		if eb.isScratch {
			a.statusBar.SetMessage("Cannot save scratch buffer")
		} else if eb.buf.Filename == "" {
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

	case cmd == "qa":
		// Quit all buffers — fail if any have unsaved changes.
		var dirtyBuffers []string
		for _, buf := range a.buffers {
			if buf.buf.Dirty {
				name := buf.Filename()
				if name == "" {
					name = "[unnamed]"
				}
				dirtyBuffers = append(dirtyBuffers, name)
			}
		}
		if len(dirtyBuffers) > 0 {
			a.statusBar.SetMessage(fmt.Sprintf("Unsaved changes in %d buffer(s): %s. Use :qa! to discard.",
				len(dirtyBuffers), strings.Join(dirtyBuffers, ", ")))
		} else {
			a.quit = true
		}

	case cmd == "qa!" || cmd == "!qa":
		// Force quit all buffers, discarding any unsaved changes.
		a.quit = true

	case cmd == "wqa" || cmd == "qwa":
		// Write all dirty buffers, then quit — fail if any unnamed buffer is dirty.
		var unnamedDirty int
		var saveFailures []string
		for _, buf := range a.buffers {
			if buf.buf.Dirty {
				if buf.buf.Filename == "" {
					unnamedDirty++
				} else {
					if err := buf.buf.Save(""); err != nil {
						saveFailures = append(saveFailures, buf.Filename()+": "+err.Error())
					}
				}
			}
		}
		if unnamedDirty > 0 {
			a.statusBar.SetMessage(fmt.Sprintf("Cannot save %d unnamed buffer(s). Use :qa! to discard, or save them first.", unnamedDirty))
		} else if len(saveFailures) > 0 {
			a.statusBar.SetMessage(fmt.Sprintf("Save failed: %s", strings.Join(saveFailures, "; ")))
		} else {
			a.quit = true
		}

	case cmd == "spell":
		a.toggleSpellCheck()

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
	eb.ScheduleSpellCheck()
}

// insertNewline splits the current line at the cursor.
func (a *App) insertNewline() {
	eb := a.currentBuf()
	eb.undo.PushInsertLine(eb.cursorLine, eb.cursorCol, eb.cursorLine, eb.cursorCol)
	eb.buf.InsertNewline(eb.cursorLine, eb.cursorCol)
	eb.cursorLine++
	eb.cursorCol = 0
	eb.ScheduleSpellCheck()
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
	eb.ScheduleSpellCheck()
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

	// Check if yankBuffer contains multiple lines
	if strings.Contains(a.yankBuffer, "\n") {
		lines := strings.Split(a.yankBuffer, "\n")
		insertPos := eb.cursorLine + 1

		// Push undo operation for multi-line insert
		eb.undo.PushInsertMultipleLines(insertPos, lines, eb.cursorLine, eb.cursorCol)

		// Insert all lines at once
		newLines := make([]string, len(eb.buf.Lines)+len(lines))
		copy(newLines, eb.buf.Lines[:insertPos])
		copy(newLines[insertPos:], lines)
		copy(newLines[insertPos+len(lines):], eb.buf.Lines[insertPos:])
		eb.buf.Lines = newLines
		eb.buf.Dirty = true

		eb.cursorLine = insertPos
		eb.cursorCol = 0
	} else {
		// Single line paste
		eb.buf.InsertLine(eb.cursorLine+1, a.yankBuffer)
		eb.undo.PushInsertWholeLine(eb.cursorLine + 1)
		eb.cursorLine++
		eb.cursorCol = 0
	}

	eb.ScheduleSpellCheck()
}

func (a *App) pasteAbove() {
	if a.yankBuffer == "" {
		return
	}
	eb := a.currentBuf()

	// Check if yankBuffer contains multiple lines
	if strings.Contains(a.yankBuffer, "\n") {
		lines := strings.Split(a.yankBuffer, "\n")
		insertPos := eb.cursorLine

		// Push undo operation for multi-line insert
		eb.undo.PushInsertMultipleLines(insertPos, lines, eb.cursorLine, eb.cursorCol)

		// Insert all lines at once
		newLines := make([]string, len(eb.buf.Lines)+len(lines))
		copy(newLines, eb.buf.Lines[:insertPos])
		copy(newLines[insertPos:], lines)
		copy(newLines[insertPos+len(lines):], eb.buf.Lines[insertPos:])
		eb.buf.Lines = newLines
		eb.buf.Dirty = true

		eb.cursorLine = insertPos
		eb.cursorCol = 0
	} else {
		// Single line paste
		eb.buf.InsertLine(eb.cursorLine, a.yankBuffer)
		eb.undo.PushInsertWholeLine(eb.cursorLine)
		eb.cursorCol = 0
	}

	eb.ScheduleSpellCheck()
}

func (a *App) undoAction() {
	eb := a.currentBuf()
	line, col, ok := eb.undo.Undo(eb.buf)
	if ok {
		eb.cursorLine = line
		eb.cursorCol = col
		eb.ScheduleSpellCheck()
	}
}

func (a *App) redoAction() {
	eb := a.currentBuf()
	line, col, ok := eb.undo.Redo(eb.buf)
	if ok {
		eb.cursorLine = line
		eb.cursorCol = col
		eb.ScheduleSpellCheck()
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
	eb.ScheduleSpellCheck()
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
	eb.ScheduleSpellCheck()
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

// jumpToNextSpellError moves the cursor to the next spelling error, wrapping around if needed.
func (a *App) jumpToNextSpellError() {
	eb := a.currentBuf()
	if len(eb.spellErrors) == 0 {
		a.statusBar.SetMessage("No spelling errors")
		return
	}

	// Find the next error after the current cursor position.
	for _, err := range eb.spellErrors {
		if err.Line > eb.cursorLine || (err.Line == eb.cursorLine && err.StartCol > eb.cursorCol) {
			eb.cursorLine = err.Line
			eb.cursorCol = err.StartCol
			return
		}
	}

	// Wrap around to the first error.
	eb.cursorLine = eb.spellErrors[0].Line
	eb.cursorCol = eb.spellErrors[0].StartCol
}

// jumpToPrevSpellError moves the cursor to the previous spelling error, wrapping around if needed.
func (a *App) jumpToPrevSpellError() {
	eb := a.currentBuf()
	if len(eb.spellErrors) == 0 {
		a.statusBar.SetMessage("No spelling errors")
		return
	}

	// Find the previous error before the current cursor position (iterate backwards).
	for i := len(eb.spellErrors) - 1; i >= 0; i-- {
		err := eb.spellErrors[i]
		if err.Line < eb.cursorLine || (err.Line == eb.cursorLine && err.StartCol < eb.cursorCol) {
			eb.cursorLine = err.Line
			eb.cursorCol = err.StartCol
			return
		}
	}

	// Wrap around to the last error.
	lastErr := eb.spellErrors[len(eb.spellErrors)-1]
	eb.cursorLine = lastErr.Line
	eb.cursorCol = lastErr.StartCol
}

// jumpToNextWord moves the cursor to the start of the next word, wrapping around if needed.
func (a *App) jumpToNextWord() {
	eb := a.currentBuf()

	// Find all word boundaries in the buffer
	boundaries := FindWordBoundaries(eb.buf)

	if len(boundaries) == 0 {
		return
	}

	// Find the next word after the current cursor position.
	for _, boundary := range boundaries {
		if boundary.Line > eb.cursorLine || (boundary.Line == eb.cursorLine && boundary.StartCol > eb.cursorCol) {
			eb.cursorLine = boundary.Line
			eb.cursorCol = boundary.StartCol
			return
		}
	}

	// Wrap around to the first word.
	eb.cursorLine = boundaries[0].Line
	eb.cursorCol = boundaries[0].StartCol
}

// jumpToPrevWord moves the cursor to the start of the previous word, wrapping around if needed.
func (a *App) jumpToPrevWord() {
	eb := a.currentBuf()

	// Find all word boundaries in the buffer
	boundaries := FindWordBoundaries(eb.buf)

	if len(boundaries) == 0 {
		return
	}

	// Find the previous word before the current cursor position (iterate backwards).
	for i := len(boundaries) - 1; i >= 0; i-- {
		boundary := boundaries[i]
		if boundary.Line < eb.cursorLine || (boundary.Line == eb.cursorLine && boundary.StartCol < eb.cursorCol) {
			eb.cursorLine = boundary.Line
			eb.cursorCol = boundary.StartCol
			return
		}
	}

	// Wrap around to the last word.
	lastBoundary := boundaries[len(boundaries)-1]
	eb.cursorLine = lastBoundary.Line
	eb.cursorCol = lastBoundary.StartCol
}

// activateSearch performs a case-insensitive search for the query and jumps to the first match.
func (a *App) activateSearch(query string) {
	eb := a.currentBuf()

	if query == "" {
		return
	}

	// Clear previous search state
	eb.searchMatches = nil
	eb.searchQuery = query
	eb.searchCurrentIdx = -1

	// Convert query to lowercase for case-insensitive matching
	queryLower := strings.ToLower(query)
	queryRunes := []rune(queryLower)

	// Search all lines for matches
	for lineIdx := 0; lineIdx < len(eb.buf.Lines); lineIdx++ {
		line := eb.buf.Lines[lineIdx]
		lineRunes := []rune(line)
		lineLower := []rune(strings.ToLower(line))

		// Check for substring match at each position
		for col := 0; col <= len(lineRunes)-len(queryRunes); col++ {
			match := true
			for i := 0; i < len(queryRunes); i++ {
				if lineLower[col+i] != queryRunes[i] {
					match = false
					break
				}
			}
			if match {
				eb.searchMatches = append(eb.searchMatches, SearchMatch{
					Line:     lineIdx,
					StartCol: col,
					EndCol:   col + len(queryRunes),
				})
			}
		}
	}

	// If no matches found, show message and return
	if len(eb.searchMatches) == 0 {
		a.statusBar.SetMessage("No matches found")
		eb.searchActive = false
		return
	}

	// Activate search and jump to nearest match
	eb.searchActive = true
	a.jumpToNearestMatch(true)
}

// clearSearch clears the search state and highlighting.
func (a *App) clearSearch() {
	eb := a.currentBuf()
	eb.searchActive = false
	eb.searchQuery = ""
	eb.searchMatches = nil
	eb.searchCurrentIdx = -1
}

// jumpToNextMatch moves to the next search match with wraparound.
func (a *App) jumpToNextMatch() {
	eb := a.currentBuf()
	if !eb.searchActive || len(eb.searchMatches) == 0 {
		return
	}

	// Move to next match
	eb.searchCurrentIdx++
	if eb.searchCurrentIdx >= len(eb.searchMatches) {
		eb.searchCurrentIdx = 0 // Wrap to first
	}

	// Jump to the match
	match := eb.searchMatches[eb.searchCurrentIdx]
	eb.cursorLine = match.Line
	eb.cursorCol = match.StartCol
}

// jumpToPrevMatch moves to the previous search match with wraparound.
func (a *App) jumpToPrevMatch() {
	eb := a.currentBuf()
	if !eb.searchActive || len(eb.searchMatches) == 0 {
		return
	}

	// Move to previous match
	eb.searchCurrentIdx--
	if eb.searchCurrentIdx < 0 {
		eb.searchCurrentIdx = len(eb.searchMatches) - 1 // Wrap to last
	}

	// Jump to the match
	match := eb.searchMatches[eb.searchCurrentIdx]
	eb.cursorLine = match.Line
	eb.cursorCol = match.StartCol
}

// jumpToNearestMatch finds the closest match from the current cursor position.
// If forward is true, finds the first match at or after cursor; otherwise finds the last match before cursor.
func (a *App) jumpToNearestMatch(forward bool) {
	eb := a.currentBuf()
	if len(eb.searchMatches) == 0 {
		return
	}

	if forward {
		// Find first match at or after cursor
		for i, match := range eb.searchMatches {
			if match.Line > eb.cursorLine || (match.Line == eb.cursorLine && match.StartCol >= eb.cursorCol) {
				eb.searchCurrentIdx = i
				eb.cursorLine = match.Line
				eb.cursorCol = match.StartCol
				return
			}
		}
		// No match after cursor, wrap to first
		eb.searchCurrentIdx = 0
		match := eb.searchMatches[0]
		eb.cursorLine = match.Line
		eb.cursorCol = match.StartCol
	} else {
		// Find last match before cursor
		for i := len(eb.searchMatches) - 1; i >= 0; i-- {
			match := eb.searchMatches[i]
			if match.Line < eb.cursorLine || (match.Line == eb.cursorLine && match.StartCol < eb.cursorCol) {
				eb.searchCurrentIdx = i
				eb.cursorLine = match.Line
				eb.cursorCol = match.StartCol
				return
			}
		}
		// No match before cursor, wrap to last
		eb.searchCurrentIdx = len(eb.searchMatches) - 1
		match := eb.searchMatches[eb.searchCurrentIdx]
		eb.cursorLine = match.Line
		eb.cursorCol = match.StartCol
	}
}

// ensureScratchBuffer ensures the scratch buffer exists and returns its index.
func (a *App) ensureScratchBuffer() int {
	// Check if scratch buffer already exists.
	for i, eb := range a.buffers {
		if eb.isScratch {
			return i
		}
	}

	// Create new scratch buffer.
	scratch := NewEditorBuffer("")
	scratch.isScratch = true
	scratch.buf.Lines = []string{""} // Start with one empty line
	a.buffers = append(a.buffers, scratch)
	return len(a.buffers) - 1
}

// jumpToScratch switches to the scratch buffer, creating it if needed.
func (a *App) jumpToScratch() {
	idx := a.ensureScratchBuffer()
	a.currentBuffer = idx
}

// sendCurrentLineToScratch sends the current line to the scratch buffer.
func (a *App) sendCurrentLineToScratch() {
	eb := a.currentBuf()
	line := eb.buf.Lines[eb.cursorLine]
	a.appendToScratch(line)
	a.statusBar.SetMessage("Sent line to scratch")
}

// appendToScratch appends content to the scratch buffer with newline separators.
func (a *App) appendToScratch(content string) {
	idx := a.ensureScratchBuffer()
	scratch := a.buffers[idx]

	if len(scratch.buf.Lines) == 1 && scratch.buf.Lines[0] == "" {
		// First entry - no separator, just replace empty line
		scratch.buf.Lines[0] = content
	} else {
		// Append with newline separator
		scratch.buf.Lines = append(scratch.buf.Lines, content)
	}
}

// getSelectionRange returns the start and end line of the current selection, ensuring start <= end.
func (a *App) getSelectionRange() (int, int) {
	start := a.lineSelectAnchor
	end := a.currentBuf().cursorLine
	if start > end {
		start, end = end, start
	}
	return start, end
}

// yankSelectedLines yanks the selected lines to the yank buffer.
func (a *App) yankSelectedLines() {
	eb := a.currentBuf()
	start, end := a.getSelectionRange()
	lines := eb.buf.Lines[start : end+1]
	a.yankBuffer = strings.Join(lines, "\n")
	a.statusBar.SetMessage(fmt.Sprintf("Yanked %d line(s)", end-start+1))
}

// deleteSelectedLines deletes the selected lines and cuts them to the yank buffer.
func (a *App) deleteSelectedLines() {
	eb := a.currentBuf()
	start, end := a.getSelectionRange()
	lines := make([]string, end-start+1)
	copy(lines, eb.buf.Lines[start:end+1])
	a.yankBuffer = strings.Join(lines, "\n") // Cut semantics

	// Push undo operation before modifying buffer
	eb.undo.PushDeleteMultipleLines(start, end, lines, eb.cursorLine, eb.cursorCol)

	// Check if deleting entire buffer
	if start == 0 && end == len(eb.buf.Lines)-1 {
		eb.buf.Lines = []string{""} // Deleting entire buffer leaves one empty line
	} else {
		eb.buf.Lines = append(eb.buf.Lines[:start], eb.buf.Lines[end+1:]...)
	}

	eb.buf.Dirty = true
	eb.cursorLine = start
	if eb.cursorLine >= len(eb.buf.Lines) {
		eb.cursorLine = len(eb.buf.Lines) - 1
	}
	eb.cursorCol = 0
	eb.ScheduleSpellCheck()

	a.statusBar.SetMessage(fmt.Sprintf("Deleted %d line(s)", end-start+1))
}

// sendSelectedLinesToScratch sends the selected lines to the scratch buffer.
func (a *App) sendSelectedLinesToScratch() {
	eb := a.currentBuf()
	start, end := a.getSelectionRange()
	lines := eb.buf.Lines[start : end+1]
	content := strings.Join(lines, "\n")

	a.appendToScratch(content)
	a.statusBar.SetMessage(fmt.Sprintf("Sent %d line(s) to scratch", end-start+1))
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

	statusLeft := a.statusBar.FormatLeft(eb.Filename(), eb.IsDirty(), bufferInfo, eb.SpellErrorCount(), eb.isScratch)
	statusRight := a.statusBar.FormatRight(a.mode, eb.WordCount(), eb.SpellErrorCount(), eb.searchActive, eb.searchCurrentIdx, len(eb.searchMatches))

	// Get selection range for line-select mode
	selectionStart, selectionEnd := -1, -1
	if a.mode == ModeLineSelect {
		selectionStart, selectionEnd = a.getSelectionRange()
	}

	frame := a.renderer.RenderFrame(displayLines, a.viewport, eb.scrollOffset, cursorDL, cursorDC, statusLeft, statusRight, eb.highlighter, eb.spellErrors, a.mode, selectionStart, selectionEnd, eb.searchActive, eb.searchMatches, eb.searchCurrentIdx)

	// Render picker overlay if active.
	if a.picker.Active {
		frame += a.renderer.RenderPicker(a.buffers, a.picker, a.currentBuffer, a.viewport)
	}

	// Render outline overlay if active.
	if a.outline.Active {
		frame += a.renderer.RenderOutline(a.outline, a.viewport)
	}

	// Render browser overlay if active.
	if a.browser.Active {
		frame += a.renderer.RenderBrowser(a.browser, a.viewport)
	}

	// Render column adjuster overlay if active.
	if a.columnAdjust.Active {
		frame += a.renderer.RenderColumnAdjust(a.columnAdjust, a.viewport)
	}

	os.Stdout.WriteString("\x1b[?2026h" + frame + "\x1b[?2026l")
}

// toggleSpellCheck toggles spell checking on/off globally.
func (a *App) toggleSpellCheck() {
	a.spellCheckEnabled = !a.spellCheckEnabled

	if a.spellCheckEnabled {
		// Turning on: run spell check on all appropriate buffers.
		for _, eb := range a.buffers {
			if eb.ShouldSpellCheck() {
				eb.spellErrors = nil
				for i := 0; i < len(eb.buf.Lines); i++ {
					lineErrors := a.spellChecker.CheckLine(i, eb.buf.Lines[i])
					eb.spellErrors = append(eb.spellErrors, lineErrors...)
				}
			}
		}
		a.statusBar.SetMessage("Spell check enabled")
	} else {
		// Turning off: clear all spell errors.
		for _, eb := range a.buffers {
			eb.spellErrors = nil
		}
		a.statusBar.SetMessage("Spell check disabled")
	}
}

func formatBufferInfo(current, total int) string {
	return fmt.Sprintf("[%d/%d]", current, total)
}
