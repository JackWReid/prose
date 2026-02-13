package main

// Picker manages the buffer-switching overlay state.
type Picker struct {
	Active   bool
	Selected int
}

// Show activates the picker with the given buffer pre-selected.
func (p *Picker) Show(currentIndex int) {
	p.Active = true
	p.Selected = currentIndex
}

// Hide deactivates the picker.
func (p *Picker) Hide() {
	p.Active = false
}

// MoveUp moves the selection up, clamping at 0.
func (p *Picker) MoveUp() {
	if p.Selected > 0 {
		p.Selected--
	}
}

// MoveDown moves the selection down, clamping at max-1.
func (p *Picker) MoveDown(max int) {
	if p.Selected < max-1 {
		p.Selected++
	}
}
