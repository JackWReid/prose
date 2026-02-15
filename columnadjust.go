package main

// ColumnAdjust manages the column width adjustment overlay state.
type ColumnAdjust struct {
	Active    bool
	Width     int // Current adjusted width
	OrigWidth int // Width before opening (for cancel/restore)
}

// Show activates the column adjuster with the current width.
func (c *ColumnAdjust) Show(currentWidth int) {
	c.Active = true
	c.Width = currentWidth
	c.OrigWidth = currentWidth
}

// Hide deactivates the column adjuster.
func (c *ColumnAdjust) Hide() {
	c.Active = false
}

// Increase bumps the width by 1, clamped to maxWidth.
func (c *ColumnAdjust) Increase(maxWidth int) {
	if c.Width < maxWidth {
		c.Width++
	}
}

// Decrease shrinks the width by 1, clamped to a minimum of 20.
func (c *ColumnAdjust) Decrease() {
	if c.Width > 20 {
		c.Width--
	}
}
