package main

import "testing"

func TestColumnAdjustShowHide(t *testing.T) {
	ca := &ColumnAdjust{}
	ca.Show(60)
	if !ca.Active {
		t.Error("should be active after Show")
	}
	if ca.Width != 60 {
		t.Errorf("Width = %d, want 60", ca.Width)
	}
	if ca.OrigWidth != 60 {
		t.Errorf("OrigWidth = %d, want 60", ca.OrigWidth)
	}
	ca.Hide()
	if ca.Active {
		t.Error("should be inactive after Hide")
	}
}

func TestColumnAdjustIncrease(t *testing.T) {
	ca := &ColumnAdjust{Active: true, Width: 60}
	ca.Increase(200)
	if ca.Width != 61 {
		t.Errorf("Width = %d, want 61", ca.Width)
	}
}

func TestColumnAdjustIncreaseClamp(t *testing.T) {
	ca := &ColumnAdjust{Active: true, Width: 100}
	ca.Increase(100)
	if ca.Width != 100 {
		t.Errorf("Width = %d, want 100 (clamped at max)", ca.Width)
	}
}

func TestColumnAdjustDecrease(t *testing.T) {
	ca := &ColumnAdjust{Active: true, Width: 60}
	ca.Decrease()
	if ca.Width != 59 {
		t.Errorf("Width = %d, want 59", ca.Width)
	}
}

func TestColumnAdjustDecreaseClamp(t *testing.T) {
	ca := &ColumnAdjust{Active: true, Width: 20}
	ca.Decrease()
	if ca.Width != 20 {
		t.Errorf("Width = %d, want 20 (clamped at min)", ca.Width)
	}
}

func TestColumnAdjustCancelRestoresOriginal(t *testing.T) {
	ca := &ColumnAdjust{}
	ca.Show(60)
	ca.Increase(200)
	ca.Increase(200)
	if ca.Width != 62 {
		t.Errorf("Width = %d, want 62", ca.Width)
	}
	// Simulate cancel: restore original width.
	ca.Width = ca.OrigWidth
	ca.Hide()
	if ca.Width != 60 {
		t.Errorf("after cancel Width = %d, want 60", ca.Width)
	}
}

func TestViewportTargetColWidth(t *testing.T) {
	vp := NewViewport(200, 50)
	if vp.TargetColWidth != 60 {
		t.Errorf("TargetColWidth = %d, want 60", vp.TargetColWidth)
	}
	if vp.ColWidth != 60 {
		t.Errorf("ColWidth = %d, want 60", vp.ColWidth)
	}

	// Change target width.
	vp.TargetColWidth = 80
	vp.recalcLayout()
	if vp.ColWidth != 80 {
		t.Errorf("ColWidth = %d, want 80 after target change", vp.ColWidth)
	}
	if vp.LeftMargin != 60 {
		t.Errorf("LeftMargin = %d, want 60", vp.LeftMargin)
	}
}

func TestViewportTargetColWidthNarrow(t *testing.T) {
	vp := NewViewport(50, 20)
	vp.TargetColWidth = 80
	vp.recalcLayout()
	// Terminal is narrower than target, so ColWidth should be terminal width.
	if vp.ColWidth != 50 {
		t.Errorf("ColWidth = %d, want 50 (clamped to terminal)", vp.ColWidth)
	}
	if vp.LeftMargin != 0 {
		t.Errorf("LeftMargin = %d, want 0", vp.LeftMargin)
	}
}
