package main

import "testing"

func TestPickerShowHide(t *testing.T) {
	p := &Picker{}
	p.Show(2)
	if !p.Active {
		t.Error("picker should be active after Show")
	}
	if p.Selected != 2 {
		t.Errorf("Selected = %d, want 2", p.Selected)
	}
	p.Hide()
	if p.Active {
		t.Error("picker should be inactive after Hide")
	}
}

func TestPickerMoveUp(t *testing.T) {
	p := &Picker{Active: true, Selected: 2}
	p.MoveUp()
	if p.Selected != 1 {
		t.Errorf("Selected = %d, want 1", p.Selected)
	}
	p.MoveUp()
	if p.Selected != 0 {
		t.Errorf("Selected = %d, want 0", p.Selected)
	}
	// Should clamp at 0.
	p.MoveUp()
	if p.Selected != 0 {
		t.Errorf("Selected = %d, want 0 (clamped)", p.Selected)
	}
}

func TestPickerMoveDown(t *testing.T) {
	p := &Picker{Active: true, Selected: 0}
	p.MoveDown(3) // max 3 items
	if p.Selected != 1 {
		t.Errorf("Selected = %d, want 1", p.Selected)
	}
	p.MoveDown(3)
	if p.Selected != 2 {
		t.Errorf("Selected = %d, want 2", p.Selected)
	}
	// Should clamp at max-1.
	p.MoveDown(3)
	if p.Selected != 2 {
		t.Errorf("Selected = %d, want 2 (clamped)", p.Selected)
	}
}

func TestPickerShowPreselectsCurrent(t *testing.T) {
	p := &Picker{}
	p.Show(0)
	if p.Selected != 0 {
		t.Errorf("Selected = %d, want 0", p.Selected)
	}
	p.Hide()
	p.Show(5)
	if p.Selected != 5 {
		t.Errorf("Selected = %d, want 5", p.Selected)
	}
}
