package main

import (
	"os"
	"path/filepath"
	"sort"
)

// Browser manages the directory browser overlay state.
type Browser struct {
	Active       bool
	Items        []BrowserItem
	Selected     int
	ScrollOffset int
	CurrentDir   string
}

// BrowserItem represents a file or directory entry.
type BrowserItem struct {
	Name  string
	Path  string // Absolute path
	IsDir bool
}

// Show activates the browser and reads the given directory.
func (b *Browser) Show(directory string) error {
	// Resolve to absolute path.
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return err
	}

	// Read directory contents.
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}

	// Convert to BrowserItems.
	items := make([]BrowserItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, BrowserItem{
			Name:  entry.Name(),
			Path:  filepath.Join(absDir, entry.Name()),
			IsDir: entry.IsDir(),
		})
	}

	// Sort: directories first (alphabetically), then files (alphabetically).
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir // Dirs before files
		}
		return items[i].Name < items[j].Name
	})

	b.Active = true
	b.Items = items
	b.Selected = 0
	b.ScrollOffset = 0
	b.CurrentDir = absDir

	return nil
}

// Hide deactivates the browser.
func (b *Browser) Hide() {
	b.Active = false
	b.Items = nil
	b.Selected = 0
	b.ScrollOffset = 0
	b.CurrentDir = ""
}

// MoveUp moves the selection up, adjusting scroll offset if needed.
func (b *Browser) MoveUp() {
	if b.Selected > 0 {
		b.Selected--
		// Adjust scroll offset if selection moves above visible window.
		if b.Selected < b.ScrollOffset {
			b.ScrollOffset = b.Selected
		}
	}
}

// MoveDown moves the selection down, adjusting scroll offset if needed.
func (b *Browser) MoveDown() {
	if b.Selected < len(b.Items)-1 {
		b.Selected++
	}
}

// VisibleItems returns the slice of items currently visible given a max height.
func (b *Browser) VisibleItems(maxHeight int) []BrowserItem {
	if len(b.Items) == 0 {
		return nil
	}

	// Ensure selection is within bounds.
	if b.Selected >= len(b.Items) {
		b.Selected = len(b.Items) - 1
	}

	// Adjust scroll offset to keep selection visible.
	if b.Selected < b.ScrollOffset {
		b.ScrollOffset = b.Selected
	}
	if b.Selected >= b.ScrollOffset+maxHeight {
		b.ScrollOffset = b.Selected - maxHeight + 1
	}

	// Clamp scroll offset.
	if b.ScrollOffset < 0 {
		b.ScrollOffset = 0
	}
	maxScroll := len(b.Items) - maxHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if b.ScrollOffset > maxScroll {
		b.ScrollOffset = maxScroll
	}

	// Return visible slice.
	start := b.ScrollOffset
	end := b.ScrollOffset + maxHeight
	if end > len(b.Items) {
		end = len(b.Items)
	}

	return b.Items[start:end]
}

// SelectedItem returns the currently selected item, or nil if none.
func (b *Browser) SelectedItem() *BrowserItem {
	if len(b.Items) == 0 || b.Selected < 0 || b.Selected >= len(b.Items) {
		return nil
	}
	return &b.Items[b.Selected]
}
