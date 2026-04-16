package tui

// ViewMode represents the current display mode for the session table
type ViewMode int

const (
	// ViewCompact shows one line per session with all info in columns
	ViewCompact ViewMode = iota
	// ViewExpanded shows two lines per session - info on first, request on second
	ViewExpanded
	// ViewMini is a super compact mode for small terminals - minimal headers, no logo
	ViewMini
)

// String returns the display name for the view mode
func (v ViewMode) String() string {
	switch v {
	case ViewCompact:
		return "Compact"
	case ViewExpanded:
		return "Expanded"
	case ViewMini:
		return "Mini"
	default:
		return "Unknown"
	}
}

// Next cycles to the next view mode
func (v ViewMode) Next() ViewMode {
	switch v {
	case ViewCompact:
		return ViewExpanded
	case ViewExpanded:
		return ViewMini
	case ViewMini:
		return ViewCompact
	default:
		return ViewCompact
	}
}

// Column represents a displayable column in the table
type Column int

const (
	ColNum Column = iota
	ColName
	ColStatus
	ColTime
	ColModel
	ColCtx
	ColGit
	ColDir
	ColRequest
)

// ColumnDef defines a column's properties
type ColumnDef struct {
	ID       Column
	Header   string
	MinWidth int // Minimum width to display this column
	Width    int // Actual width when displayed
	Priority int // Lower = more important, hidden last
}

// DefaultColumns defines all available columns with their properties
var DefaultColumns = []ColumnDef{
	{ID: ColNum, Header: "#", Width: 3, MinWidth: 0, Priority: 0},
	{ID: ColName, Header: "SESSION", Width: 18, MinWidth: 0, Priority: 1},
	{ID: ColStatus, Header: "STATUS", Width: 8, MinWidth: 0, Priority: 2},
	{ID: ColTime, Header: "TIME", Width: 9, MinWidth: 85, Priority: 3},
	{ID: ColModel, Header: "AGENT", Width: 7, MinWidth: 100, Priority: 6},
	{ID: ColGit, Header: "GIT", Width: 20, MinWidth: 100, Priority: 5},
	{ID: ColDir, Header: "DIRECTORY", Width: 25, MinWidth: 80, Priority: 4},
	{ID: ColRequest, Header: "REQUEST", Width: 0, MinWidth: 0, Priority: 0}, // Flexible width
}

// MiniColumns defines super compact columns for small terminals
// Uses 1-2 character headers and minimal widths
var MiniColumns = []ColumnDef{
	{ID: ColNum, Header: "#", Width: 2, MinWidth: 0, Priority: 0},
	{ID: ColName, Header: "SESS", Width: 12, MinWidth: 0, Priority: 1},
	{ID: ColStatus, Header: "ST", Width: 5, MinWidth: 0, Priority: 2},
	{ID: ColTime, Header: "TIME", Width: 7, MinWidth: 60, Priority: 3},
	{ID: ColDir, Header: "DIR", Width: 15, MinWidth: 70, Priority: 4},
	{ID: ColRequest, Header: "REQ", Width: 0, MinWidth: 0, Priority: 0},
}

// MiniSeparator is a thinner separator for mini mode
const MiniSeparator = "│"

// MiniSeparatorWidth is 1 character
const MiniSeparatorWidth = 1

// ColumnSeparator is the string used between columns
const ColumnSeparator = " │ "

// SeparatorWidth is the rendered width of the column separator
const SeparatorWidth = 3

// shrinkMinWidths defines minimum widths for columns that can shrink before being hidden
var shrinkMinWidths = map[Column]int{
	ColName: 12,
	ColDir:  15,
	ColGit:  10,
}

// GetVisibleColumns returns columns that fit in the given width, sorted by priority
func GetVisibleColumns(width int) []ColumnDef {
	visible := make([]ColumnDef, 0, len(DefaultColumns))

	// Calculate total fixed width needed
	fixedWidth := 0
	for _, col := range DefaultColumns {
		if col.ID != ColRequest {
			fixedWidth += col.Width + SeparatorWidth
		}
	}

	// If we have enough space at full widths, show everything
	if width >= fixedWidth+20 {
		for _, col := range DefaultColumns {
			visible = append(visible, col)
		}
		remaining := width - fixedWidth - 4
		if remaining < 20 {
			remaining = 20
		}
		for i := range visible {
			if visible[i].ID == ColRequest {
				visible[i].Width = remaining
			}
		}
		return visible
	}

	// Phase 1: Try shrinking flexible columns before hiding any
	columnsToConsider := make([]ColumnDef, len(DefaultColumns))
	copy(columnsToConsider, DefaultColumns)

	// Calculate how much space we need to reclaim
	minRequestWidth := 15
	usedWidth := 4 // base padding
	for _, col := range columnsToConsider {
		if col.ID != ColRequest {
			usedWidth += col.Width + SeparatorWidth
		}
	}
	usedWidth += minRequestWidth + SeparatorWidth
	deficit := usedWidth - width

	if deficit > 0 {
		// Shrink columns with shrinkMinWidths, largest shrink potential first
		for deficit > 0 {
			bestIdx := -1
			bestShrink := 0
			for i, col := range columnsToConsider {
				if minW, ok := shrinkMinWidths[col.ID]; ok {
					available := col.Width - minW
					if available > bestShrink {
						bestShrink = available
						bestIdx = i
					}
				}
			}
			if bestIdx == -1 || bestShrink == 0 {
				break
			}
			delta := bestShrink
			if delta > deficit {
				delta = deficit
			}
			columnsToConsider[bestIdx].Width -= delta
			deficit -= delta
		}
	}

	// Phase 2: If still doesn't fit, hide columns by priority
	for deficit > 0 {
		numCols := 0
		for _, col := range columnsToConsider {
			if col.ID != ColRequest {
				numCols++
			}
		}
		if numCols <= 3 {
			break
		}

		maxPriority := -1
		removeIdx := -1
		for i, col := range columnsToConsider {
			if col.ID == ColRequest || col.ID == ColNum || col.ID == ColName || col.ID == ColStatus {
				continue
			}
			if col.Priority > maxPriority {
				maxPriority = col.Priority
				removeIdx = i
			}
		}
		if removeIdx == -1 {
			break
		}

		removed := columnsToConsider[removeIdx]
		columnsToConsider = append(columnsToConsider[:removeIdx], columnsToConsider[removeIdx+1:]...)
		deficit -= removed.Width + SeparatorWidth
	}

	// Calculate final request width
	usedWidth = 4
	for _, col := range columnsToConsider {
		if col.ID != ColRequest {
			usedWidth += col.Width + SeparatorWidth
		}
	}
	remaining := width - usedWidth - SeparatorWidth
	if remaining < minRequestWidth {
		remaining = minRequestWidth
	}

	for _, col := range columnsToConsider {
		if col.ID == ColRequest {
			col.Width = remaining
		}
		visible = append(visible, col)
	}

	return visible
}

// HasColumn checks if a column ID is in the visible columns
func HasColumn(columns []ColumnDef, id Column) bool {
	for _, col := range columns {
		if col.ID == id {
			return true
		}
	}
	return false
}

// GetColumnWidth returns the width of a specific column, or 0 if not visible
func GetColumnWidth(columns []ColumnDef, id Column) int {
	for _, col := range columns {
		if col.ID == id {
			return col.Width
		}
	}
	return 0
}
