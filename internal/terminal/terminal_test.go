package terminal

import "testing"

func TestParseKeyRune(t *testing.T) {
	k := parseKey([]byte{'a'})
	if k.Type != KeyRune || k.Rune != 'a' {
		t.Errorf("expected rune 'a', got type=%d rune=%c", k.Type, k.Rune)
	}
}

func TestParseKeyEscape(t *testing.T) {
	k := parseKey([]byte{27})
	if k.Type != KeyEscape {
		t.Errorf("expected escape, got type=%d", k.Type)
	}
}

func TestParseKeyEnter(t *testing.T) {
	k := parseKey([]byte{13})
	if k.Type != KeyEnter {
		t.Errorf("expected enter, got type=%d", k.Type)
	}
}

func TestParseKeyBackspace(t *testing.T) {
	k := parseKey([]byte{127})
	if k.Type != KeyBackspace {
		t.Errorf("expected backspace (127), got type=%d", k.Type)
	}
	k = parseKey([]byte{8})
	if k.Type != KeyBackspace {
		t.Errorf("expected backspace (8), got type=%d", k.Type)
	}
}

func TestParseKeyCtrlZ(t *testing.T) {
	k := parseKey([]byte{26})
	if k.Type != KeyCtrlZ {
		t.Errorf("expected ctrl-z, got type=%d", k.Type)
	}
}

func TestParseKeyCtrlY(t *testing.T) {
	k := parseKey([]byte{25})
	if k.Type != KeyCtrlY {
		t.Errorf("expected ctrl-y, got type=%d", k.Type)
	}
}

func TestParseKeyCtrlR(t *testing.T) {
	k := parseKey([]byte{18})
	if k.Type != KeyCtrlR {
		t.Errorf("expected ctrl-r, got type=%d", k.Type)
	}
}

func TestParseKeyArrows(t *testing.T) {
	tests := []struct {
		seq      []byte
		expected int
	}{
		{[]byte{27, '[', 'A'}, KeyUp},
		{[]byte{27, '[', 'B'}, KeyDown},
		{[]byte{27, '[', 'C'}, KeyRight},
		{[]byte{27, '[', 'D'}, KeyLeft},
	}
	for _, tc := range tests {
		k := parseKey(tc.seq)
		if k.Type != tc.expected {
			t.Errorf("seq %v: expected type %d, got %d", tc.seq, tc.expected, k.Type)
		}
	}
}

func TestParseKeyEmpty(t *testing.T) {
	k := parseKey([]byte{})
	if k.Type != KeyUnknown {
		t.Errorf("expected unknown for empty input, got type=%d", k.Type)
	}
}

func TestParseKeyControlChar(t *testing.T) {
	// Control char that isn't specifically handled.
	k := parseKey([]byte{1}) // Ctrl+A
	if k.Type != KeyUnknown {
		t.Errorf("expected unknown for ctrl-a, got type=%d", k.Type)
	}
}

func TestDecodeUTF8(t *testing.T) {
	// ASCII
	if r := decodeUTF8([]byte{'A'}); r != 'A' {
		t.Errorf("ASCII: got %c", r)
	}
	// 2-byte: é (U+00E9) = 0xC3 0xA9
	if r := decodeUTF8([]byte{0xC3, 0xA9}); r != 'é' {
		t.Errorf("2-byte: got %c (%x)", r, r)
	}
	// 3-byte: 日 (U+65E5) = 0xE6 0x97 0xA5
	if r := decodeUTF8([]byte{0xE6, 0x97, 0xA5}); r != '日' {
		t.Errorf("3-byte: got %c (%x)", r, r)
	}
	// Empty
	if r := decodeUTF8([]byte{}); r != 0 {
		t.Errorf("empty: got %x", r)
	}
	// Invalid continuation byte
	if r := decodeUTF8([]byte{0x80}); r != 0xFFFD {
		t.Errorf("invalid: got %x", r)
	}
}

func TestParseKeyMultibyteUTF8(t *testing.T) {
	// é as multi-byte input
	k := parseKey([]byte{0xC3, 0xA9})
	if k.Type != KeyRune || k.Rune != 'é' {
		t.Errorf("expected rune é, got type=%d rune=%c", k.Type, k.Rune)
	}
}

func TestParseKeyCtrlD(t *testing.T) {
	k := parseKey([]byte{4})
	if k.Type != KeyCtrlD {
		t.Errorf("expected ctrl-d, got type=%d", k.Type)
	}
}

func TestParseKeyCtrlU(t *testing.T) {
	k := parseKey([]byte{21})
	if k.Type != KeyCtrlU {
		t.Errorf("expected ctrl-u, got type=%d", k.Type)
	}
}

func TestParseKeyHomeEnd3Byte(t *testing.T) {
	// Home: ESC [ H
	k := parseKey([]byte{27, '[', 'H'})
	if k.Type != KeyHome {
		t.Errorf("expected home (3-byte), got type=%d", k.Type)
	}
	// End: ESC [ F
	k = parseKey([]byte{27, '[', 'F'})
	if k.Type != KeyEnd {
		t.Errorf("expected end (3-byte), got type=%d", k.Type)
	}
}

func TestParseKeyCSI4Byte(t *testing.T) {
	tests := []struct {
		seq      []byte
		expected int
		name     string
	}{
		{[]byte{27, '[', '1', '~'}, KeyHome, "home"},
		{[]byte{27, '[', '3', '~'}, KeyDelete, "delete"},
		{[]byte{27, '[', '4', '~'}, KeyEnd, "end"},
		{[]byte{27, '[', '5', '~'}, KeyPgUp, "pgup"},
		{[]byte{27, '[', '6', '~'}, KeyPgDn, "pgdn"},
	}
	for _, tc := range tests {
		k := parseKey(tc.seq)
		if k.Type != tc.expected {
			t.Errorf("%s: expected type %d, got %d", tc.name, tc.expected, k.Type)
		}
	}
}

func TestParseMouseEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantOK    bool
		wantBtn   MouseButton
		wantRow   int
		wantCol   int
		wantPress bool
	}{
		{
			name:      "left button press",
			input:     []byte("\x1b[<0;10;5M"),
			wantOK:    true,
			wantBtn:   MouseLeft,
			wantRow:   5,
			wantCol:   10,
			wantPress: true,
		},
		{
			name:      "left button release",
			input:     []byte("\x1b[<0;10;5m"),
			wantOK:    true,
			wantBtn:   MouseLeft,
			wantRow:   5,
			wantCol:   10,
			wantPress: false,
		},
		{
			name:      "middle button press",
			input:     []byte("\x1b[<1;20;15M"),
			wantOK:    true,
			wantBtn:   MouseMiddle,
			wantRow:   15,
			wantCol:   20,
			wantPress: true,
		},
		{
			name:      "right button press",
			input:     []byte("\x1b[<2;30;25M"),
			wantOK:    true,
			wantBtn:   MouseRight,
			wantRow:   25,
			wantCol:   30,
			wantPress: true,
		},
		{
			name:   "invalid sequence - too short",
			input:  []byte("\x1b[<0;1M"),
			wantOK: false,
		},
		{
			name:   "invalid sequence - no ESC",
			input:  []byte("[<0;10;5M"),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mouse, ok := parseMouseEvent(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseMouseEvent() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if !tt.wantOK {
				return
			}
			if mouse.Button != tt.wantBtn {
				t.Errorf("Button = %v, want %v", mouse.Button, tt.wantBtn)
			}
			if mouse.Row != tt.wantRow {
				t.Errorf("Row = %v, want %v", mouse.Row, tt.wantRow)
			}
			if mouse.Col != tt.wantCol {
				t.Errorf("Col = %v, want %v", mouse.Col, tt.wantCol)
			}
			if mouse.Press != tt.wantPress {
				t.Errorf("Press = %v, want %v", mouse.Press, tt.wantPress)
			}
		})
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantType  int
		wantKey   int
		wantMouse bool
	}{
		{
			name:     "regular key",
			input:    []byte("a"),
			wantType: EventKey,
			wantKey:  KeyRune,
		},
		{
			name:     "escape key",
			input:    []byte{27},
			wantType: EventKey,
			wantKey:  KeyEscape,
		},
		{
			name:      "mouse event",
			input:     []byte("\x1b[<0;10;5M"),
			wantType:  EventMouse,
			wantMouse: true,
		},
		{
			name:     "arrow up",
			input:    []byte("\x1b[A"),
			wantType: EventKey,
			wantKey:  KeyUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := parseInput(tt.input)
			if event.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", event.Type, tt.wantType)
			}
			if tt.wantType == EventKey && event.Key.Type != tt.wantKey {
				t.Errorf("Key.Type = %v, want %v", event.Key.Type, tt.wantKey)
			}
			if tt.wantMouse && event.Mouse.Button == MouseUnknown {
				t.Error("Expected valid mouse event")
			}
		})
	}
}
