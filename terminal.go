package main

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// Terminal manages raw mode, alternate screen buffer, and terminal dimensions.
type Terminal struct {
	oldState *term.State
	width    int
	height   int
	sigwinch chan os.Signal
}

func NewTerminal() (*Terminal, error) {
	t := &Terminal{}

	// Switch to raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	t.oldState = oldState

	// Enter alternate screen buffer.
	os.Stdout.WriteString("\x1b[?1049h")

	// Hide cursor during setup.
	os.Stdout.WriteString("\x1b[?25l")

	// Query size.
	t.width, t.height, err = term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		t.Restore()
		return nil, err
	}

	// Listen for resize signals.
	t.sigwinch = make(chan os.Signal, 1)
	signal.Notify(t.sigwinch, syscall.SIGWINCH)

	return t, nil
}

// Resize re-queries terminal dimensions. Returns true if the size changed.
func (t *Terminal) Resize() bool {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return false
	}
	changed := w != t.width || h != t.height
	t.width = w
	t.height = h
	return changed
}

// SigwinchChan returns the channel that receives SIGWINCH signals.
func (t *Terminal) SigwinchChan() <-chan os.Signal {
	return t.sigwinch
}

// Restore returns the terminal to its original state.
func (t *Terminal) Restore() {
	// Show cursor.
	os.Stdout.WriteString("\x1b[?25h")
	// Leave alternate screen buffer.
	os.Stdout.WriteString("\x1b[?1049l")
	if t.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), t.oldState)
	}
	signal.Stop(t.sigwinch)
}

// ReadKey reads a single keypress from stdin in raw mode.
// Returns a Key struct describing the input.
func (t *Terminal) ReadKey() (Key, error) {
	buf := make([]byte, 6)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		return Key{}, err
	}
	return parseKey(buf[:n]), nil
}

// Key types.
const (
	KeyRune      = iota // Normal printable character
	KeyEscape           // Escape key (standalone)
	KeyEnter            // Enter/Return
	KeyBackspace        // Backspace/Delete-backward
	KeyUp               // Arrow up
	KeyDown             // Arrow down
	KeyLeft             // Arrow left
	KeyRight            // Arrow right
	KeyCtrlZ            // Ctrl+Z
	KeyCtrlY            // Ctrl+Y
	KeyCtrlR            // Ctrl+R
	KeyCtrlD            // Ctrl+D
	KeyCtrlU            // Ctrl+U
	KeyHome             // Home
	KeyEnd              // End
	KeyDelete           // Delete/Forward-delete
	KeyPgUp             // Page Up
	KeyPgDn             // Page Down
	KeyUnknown          // Unrecognised sequence
)

type Key struct {
	Type int
	Rune rune
}

func parseKey(buf []byte) Key {
	if len(buf) == 0 {
		return Key{Type: KeyUnknown}
	}

	// Single byte.
	if len(buf) == 1 {
		b := buf[0]
		switch {
		case b == 27:
			return Key{Type: KeyEscape}
		case b == 13:
			return Key{Type: KeyEnter}
		case b == 127 || b == 8:
			return Key{Type: KeyBackspace}
		case b == 26: // Ctrl+Z
			return Key{Type: KeyCtrlZ}
		case b == 25: // Ctrl+Y
			return Key{Type: KeyCtrlY}
		case b == 18: // Ctrl+R
			return Key{Type: KeyCtrlR}
		case b == 4: // Ctrl+D
			return Key{Type: KeyCtrlD}
		case b == 21: // Ctrl+U
			return Key{Type: KeyCtrlU}
		case b >= 32 && b < 127:
			return Key{Type: KeyRune, Rune: rune(b)}
		default:
			return Key{Type: KeyUnknown}
		}
	}

	// Escape sequences.
	if buf[0] == 27 && len(buf) >= 3 && buf[1] == '[' {
		// CSI 3-byte sequences.
		switch buf[2] {
		case 'A':
			return Key{Type: KeyUp}
		case 'B':
			return Key{Type: KeyDown}
		case 'C':
			return Key{Type: KeyRight}
		case 'D':
			return Key{Type: KeyLeft}
		case 'H':
			return Key{Type: KeyHome}
		case 'F':
			return Key{Type: KeyEnd}
		}

		// CSI 4-byte sequences: ESC [ <n> ~
		if len(buf) >= 4 && buf[3] == '~' {
			switch buf[2] {
			case '1':
				return Key{Type: KeyHome}
			case '3':
				return Key{Type: KeyDelete}
			case '4':
				return Key{Type: KeyEnd}
			case '5':
				return Key{Type: KeyPgUp}
			case '6':
				return Key{Type: KeyPgDn}
			}
		}
	}

	// Multi-byte UTF-8 character.
	r := decodeUTF8(buf)
	if r >= 32 {
		return Key{Type: KeyRune, Rune: r}
	}

	return Key{Type: KeyUnknown}
}

func decodeUTF8(buf []byte) rune {
	if len(buf) == 0 {
		return 0
	}
	// Simple UTF-8 decode for 1–4 byte sequences.
	b := buf[0]
	switch {
	case b < 0x80:
		return rune(b)
	case b < 0xC0:
		return 0xFFFD
	case b < 0xE0 && len(buf) >= 2:
		return rune(b&0x1F)<<6 | rune(buf[1]&0x3F)
	case b < 0xF0 && len(buf) >= 3:
		return rune(b&0x0F)<<12 | rune(buf[1]&0x3F)<<6 | rune(buf[2]&0x3F)
	case b < 0xF8 && len(buf) >= 4:
		return rune(b&0x07)<<18 | rune(buf[1]&0x3F)<<12 | rune(buf[2]&0x3F)<<6 | rune(buf[3]&0x3F)
	}
	return 0xFFFD
}
