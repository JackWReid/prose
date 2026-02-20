package editor

// Outline manages the document outline overlay state.
type Outline struct {
	Active       bool
	Items        []OutlineItem
	Selected     int
	ScrollOffset int // For scrolling long outlines
}

// Show activates the outline with the given items.
func (o *Outline) Show(items []OutlineItem) {
	o.Active = true
	o.Items = items
	o.Selected = 0
	o.ScrollOffset = 0
}

// Hide deactivates the outline.
func (o *Outline) Hide() {
	o.Active = false
	o.Items = nil
	o.Selected = 0
	o.ScrollOffset = 0
}

// MoveUp moves the selection up, adjusting scroll offset if needed.
func (o *Outline) MoveUp() {
	if o.Selected > 0 {
		o.Selected--
		// Adjust scroll offset if selection moves above visible window.
		if o.Selected < o.ScrollOffset {
			o.ScrollOffset = o.Selected
		}
	}
}

// MoveDown moves the selection down, adjusting scroll offset if needed.
func (o *Outline) MoveDown() {
	if o.Selected < len(o.Items)-1 {
		o.Selected++
		// Adjust scroll offset if selection moves below visible window.
		// We'll calculate the max visible items in the renderer.
	}
}

// VisibleItems returns the slice of items currently visible given a max height.
func (o *Outline) VisibleItems(maxHeight int) []OutlineItem {
	if len(o.Items) == 0 {
		return nil
	}

	// Ensure selection is within bounds.
	if o.Selected >= len(o.Items) {
		o.Selected = len(o.Items) - 1
	}

	// Adjust scroll offset to keep selection visible.
	if o.Selected < o.ScrollOffset {
		o.ScrollOffset = o.Selected
	}
	if o.Selected >= o.ScrollOffset+maxHeight {
		o.ScrollOffset = o.Selected - maxHeight + 1
	}

	// Clamp scroll offset.
	if o.ScrollOffset < 0 {
		o.ScrollOffset = 0
	}
	maxScroll := len(o.Items) - maxHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if o.ScrollOffset > maxScroll {
		o.ScrollOffset = maxScroll
	}

	// Return visible slice.
	start := o.ScrollOffset
	end := o.ScrollOffset + maxHeight
	if end > len(o.Items) {
		end = len(o.Items)
	}

	return o.Items[start:end]
}
