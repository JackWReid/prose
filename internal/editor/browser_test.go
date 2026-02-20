package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserShow(t *testing.T) {
	dir := t.TempDir()
	// Create some test files and directories.
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.md"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	b := &Browser{}
	err := b.Show(dir)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if !b.Active {
		t.Error("browser should be active after Show")
	}
	if len(b.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(b.Items))
	}
	if b.Selected != 0 {
		t.Errorf("selected should start at 0, got %d", b.Selected)
	}
	if b.ScrollOffset != 0 {
		t.Errorf("scroll offset should start at 0, got %d", b.ScrollOffset)
	}
}

func TestBrowserSortsDirsFirst(t *testing.T) {
	dir := t.TempDir()
	// Create files and directories in non-alphabetical order.
	os.WriteFile(filepath.Join(dir, "zzz.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(dir, "aaa-dir"), 0755)
	os.WriteFile(filepath.Join(dir, "bbb.md"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(dir, "ccc-dir"), 0755)

	b := &Browser{}
	err := b.Show(dir)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if len(b.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(b.Items))
	}

	// Directories should come first, alphabetically.
	if !b.Items[0].IsDir || b.Items[0].Name != "aaa-dir" {
		t.Errorf("first item should be aaa-dir, got %s (isDir=%v)", b.Items[0].Name, b.Items[0].IsDir)
	}
	if !b.Items[1].IsDir || b.Items[1].Name != "ccc-dir" {
		t.Errorf("second item should be ccc-dir, got %s (isDir=%v)", b.Items[1].Name, b.Items[1].IsDir)
	}

	// Files should come after, alphabetically.
	if b.Items[2].IsDir || b.Items[2].Name != "bbb.md" {
		t.Errorf("third item should be bbb.md, got %s (isDir=%v)", b.Items[2].Name, b.Items[2].IsDir)
	}
	if b.Items[3].IsDir || b.Items[3].Name != "zzz.txt" {
		t.Errorf("fourth item should be zzz.txt, got %s (isDir=%v)", b.Items[3].Name, b.Items[3].IsDir)
	}
}

func TestBrowserHide(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(dir)

	b.Hide()

	if b.Active {
		t.Error("browser should not be active after Hide")
	}
	if len(b.Items) != 0 {
		t.Errorf("items should be cleared, got %d items", len(b.Items))
	}
	if b.CurrentDir != "" {
		t.Error("current dir should be cleared")
	}
}

func TestBrowserMoveUp(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file3.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(dir)

	// Move down twice.
	b.MoveDown()
	b.MoveDown()
	if b.Selected != 2 {
		t.Errorf("selected should be 2, got %d", b.Selected)
	}

	// Move up once.
	b.MoveUp()
	if b.Selected != 1 {
		t.Errorf("selected should be 1, got %d", b.Selected)
	}

	// Move up twice (should clamp at 0).
	b.MoveUp()
	b.MoveUp()
	if b.Selected != 0 {
		t.Errorf("selected should be clamped at 0, got %d", b.Selected)
	}
}

func TestBrowserMoveDown(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file3.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(dir)

	b.MoveDown()
	if b.Selected != 1 {
		t.Errorf("selected should be 1, got %d", b.Selected)
	}

	b.MoveDown()
	if b.Selected != 2 {
		t.Errorf("selected should be 2, got %d", b.Selected)
	}

	// Should clamp at max-1.
	b.MoveDown()
	if b.Selected != 2 {
		t.Errorf("selected should be clamped at 2, got %d", b.Selected)
	}
}

func TestBrowserVisibleItems(t *testing.T) {
	dir := t.TempDir()
	// Create 10 files.
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(dir, filepath.Base(t.TempDir())+".txt"), []byte("content"), 0644)
	}

	b := &Browser{}
	b.Show(dir)

	// Get 5 visible items.
	visible := b.VisibleItems(5)
	if len(visible) != 5 {
		t.Errorf("expected 5 visible items, got %d", len(visible))
	}

	// Move selection down to item 7.
	for i := 0; i < 7; i++ {
		b.MoveDown()
	}

	// Visible items should scroll to keep selection visible.
	visible = b.VisibleItems(5)
	if len(visible) != 5 {
		t.Errorf("expected 5 visible items after scroll, got %d", len(visible))
	}
	if b.ScrollOffset != 3 {
		t.Errorf("scroll offset should be 3 (7-5+1), got %d", b.ScrollOffset)
	}
}

func TestBrowserEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	b := &Browser{}
	err := b.Show(dir)
	if err != nil {
		t.Fatalf("Show failed on empty dir: %v", err)
	}

	if len(b.Items) != 0 {
		t.Errorf("expected 0 items in empty directory, got %d", len(b.Items))
	}

	visible := b.VisibleItems(10)
	if len(visible) != 0 {
		t.Errorf("visible items should be empty, got %d", len(visible))
	}
}

func TestBrowserNonexistentDirectory(t *testing.T) {
	b := &Browser{}
	err := b.Show("/nonexistent/directory/path")
	if err == nil {
		t.Error("Show should fail on nonexistent directory")
	}
	if b.Active {
		t.Error("browser should not be active after failed Show")
	}
}

func TestBrowserSelectedItem(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(dir)

	item := b.SelectedItem()
	if item == nil {
		t.Fatal("selected item should not be nil")
	}
	if item.Name != "file1.txt" {
		t.Errorf("selected item name should be file1.txt, got %s", item.Name)
	}

	b.MoveDown()
	item = b.SelectedItem()
	if item == nil {
		t.Fatal("selected item should not be nil after move down")
	}
	if item.Name != "file2.txt" {
		t.Errorf("selected item name should be file2.txt, got %s", item.Name)
	}
}

func TestBrowserSelectedItemEmpty(t *testing.T) {
	b := &Browser{}
	item := b.SelectedItem()
	if item != nil {
		t.Error("selected item should be nil when browser is empty")
	}
}

func TestBrowserAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0644)

	b := &Browser{}
	err := b.Show(dir)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if !filepath.IsAbs(b.CurrentDir) {
		t.Error("CurrentDir should be absolute path")
	}
	if !filepath.IsAbs(b.Items[0].Path) {
		t.Error("item Path should be absolute")
	}
}

func TestBrowserNavigateToSubdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(dir)

	// Should have the subdirectory.
	if len(b.Items) != 1 || !b.Items[0].IsDir {
		t.Fatalf("expected 1 directory, got %d items", len(b.Items))
	}

	// Navigate into subdirectory.
	err := b.Show(b.Items[0].Path)
	if err != nil {
		t.Fatalf("failed to navigate into subdirectory: %v", err)
	}

	if b.CurrentDir != subdir {
		t.Errorf("current dir should be %s, got %s", subdir, b.CurrentDir)
	}
	if len(b.Items) != 1 || b.Items[0].Name != "file.txt" {
		t.Errorf("expected file.txt in subdirectory, got %v", b.Items)
	}
}

func TestBrowserNavigateToParent(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.Mkdir(subdir, 0755)
	os.WriteFile(filepath.Join(dir, "parent.txt"), []byte("content"), 0644)

	b := &Browser{}
	b.Show(subdir)

	if b.CurrentDir != subdir {
		t.Fatalf("should start in subdirectory")
	}

	// Navigate to parent.
	parentDir := filepath.Dir(b.CurrentDir)
	err := b.Show(parentDir)
	if err != nil {
		t.Fatalf("failed to navigate to parent: %v", err)
	}

	if b.CurrentDir != dir {
		t.Errorf("current dir should be %s, got %s", dir, b.CurrentDir)
	}

	// Should have both the subdirectory and parent.txt.
	if len(b.Items) != 2 {
		t.Errorf("expected 2 items in parent directory, got %d", len(b.Items))
	}
}
