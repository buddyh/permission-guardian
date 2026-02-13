package tui

import (
	"testing"
)

func TestViewModeString(t *testing.T) {
	tests := []struct {
		mode ViewMode
		want string
	}{
		{ViewCompact, "Compact"},
		{ViewExpanded, "Expanded"},
		{ViewMini, "Mini"},
		{ViewMode(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("ViewMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestViewModeNext(t *testing.T) {
	tests := []struct {
		mode ViewMode
		want ViewMode
	}{
		{ViewCompact, ViewExpanded},
		{ViewExpanded, ViewMini},
		{ViewMini, ViewCompact},
	}

	for _, tt := range tests {
		got := tt.mode.Next()
		if got != tt.want {
			t.Errorf("ViewMode(%d).Next() = %d, want %d", tt.mode, got, tt.want)
		}
	}
}

func TestGetVisibleColumns(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		wantMin     int // Minimum columns expected
		wantMax     int // Maximum columns expected
		mustHave    []Column
		mustNotHave []Column
	}{
		{
			name:     "very wide terminal",
			width:    200,
			wantMin:  10,
			wantMax:  10,
			mustHave: []Column{ColNum, ColName, ColStatus, ColTime, ColModel, ColCtx, ColMem, ColGit, ColDir, ColRequest},
		},
		{
			name:     "normal terminal",
			width:    120,
			wantMin:  6,
			wantMax:  9,
			mustHave: []Column{ColNum, ColName, ColStatus, ColTime, ColDir, ColRequest},
		},
		{
			name:     "narrow terminal",
			width:    80,
			wantMin:  4,
			wantMax:  7,
			mustHave: []Column{ColNum, ColName, ColStatus, ColCtx, ColRequest},
		},
		{
			name:     "very narrow terminal",
			width:    60,
			wantMin:  4,
			wantMax:  5,
			mustHave: []Column{ColNum, ColName, ColStatus, ColRequest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := GetVisibleColumns(tt.width)

			if len(cols) < tt.wantMin || len(cols) > tt.wantMax {
				t.Errorf("GetVisibleColumns(%d) returned %d columns, want between %d and %d",
					tt.width, len(cols), tt.wantMin, tt.wantMax)
			}

			for _, mustHave := range tt.mustHave {
				if !HasColumn(cols, mustHave) {
					t.Errorf("GetVisibleColumns(%d) missing required column %d", tt.width, mustHave)
				}
			}

			for _, mustNotHave := range tt.mustNotHave {
				if HasColumn(cols, mustNotHave) {
					t.Errorf("GetVisibleColumns(%d) should not have column %d at this width", tt.width, mustNotHave)
				}
			}
		})
	}
}

func TestHasColumn(t *testing.T) {
	cols := []ColumnDef{
		{ID: ColNum},
		{ID: ColName},
		{ID: ColStatus},
	}

	if !HasColumn(cols, ColNum) {
		t.Error("HasColumn should return true for ColNum")
	}
	if !HasColumn(cols, ColName) {
		t.Error("HasColumn should return true for ColName")
	}
	if HasColumn(cols, ColRequest) {
		t.Error("HasColumn should return false for ColRequest")
	}
}

func TestGetColumnWidth(t *testing.T) {
	cols := []ColumnDef{
		{ID: ColNum, Width: 3},
		{ID: ColName, Width: 18},
		{ID: ColRequest, Width: 50},
	}

	if w := GetColumnWidth(cols, ColNum); w != 3 {
		t.Errorf("GetColumnWidth(ColNum) = %d, want 3", w)
	}
	if w := GetColumnWidth(cols, ColName); w != 18 {
		t.Errorf("GetColumnWidth(ColName) = %d, want 18", w)
	}
	if w := GetColumnWidth(cols, ColGit); w != 0 {
		t.Errorf("GetColumnWidth(ColGit) = %d, want 0 (not in list)", w)
	}
}
