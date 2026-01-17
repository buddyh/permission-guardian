package tui

// ViewMode represents the current display mode for the session table
type ViewMode int

const (
	// ViewCompact shows one line per session with all info in columns
	ViewCompact ViewMode = iota
	// ViewExpanded shows two lines per session - info on first, request on second
	ViewExpanded
)

// String returns the display name for the view mode
func (v ViewMode) String() string {
	switch v {
	case ViewCompact:
		return "Compact"
	case ViewExpanded:
		return "Expanded"
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
	{ID: ColTime, Header: "TIME", Width: 7, MinWidth: 85, Priority: 3},
	{ID: ColModel, Header: "MODEL", Width: 9, MinWidth: 100, Priority: 6},
	{ID: ColCtx, Header: "CTX", Width: 6, MinWidth: 90, Priority: 7},
	{ID: ColGit, Header: "GIT", Width: 14, MinWidth: 110, Priority: 5},
	{ID: ColDir, Header: "DIRECTORY", Width: 25, MinWidth: 80, Priority: 4},
	{ID: ColRequest, Header: "REQUEST", Width: 0, MinWidth: 0, Priority: 0}, // Flexible width
}

// ColumnSeparator is the string used between columns
const ColumnSeparator = " │ "

// SeparatorWidth is the rendered width of the column separator
const SeparatorWidth = 3

// GetVisibleColumns returns columns that fit in the given width, sorted by priority
func GetVisibleColumns(width int) []ColumnDef {
	// Start with all columns
	visible := make([]ColumnDef, 0, len(DefaultColumns))

	// Calculate total fixed width needed
	fixedWidth := 0
	for _, col := range DefaultColumns {
		if col.ID != ColRequest { // Request column is flexible
			fixedWidth += col.Width + SeparatorWidth
		}
	}

	// If we have enough space, show everything
	if width >= fixedWidth+20 { // 20 min for request column
		for _, col := range DefaultColumns {
			visible = append(visible, col)
		}
		// Set request column to remaining width
		remaining := width - fixedWidth - 4 // padding
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

	// Otherwise, progressively hide columns by priority (higher priority number = hide first)
	// Keep removing columns until we fit
	columnsToConsider := make([]ColumnDef, len(DefaultColumns))
	copy(columnsToConsider, DefaultColumns)

	for {
		// Calculate current width
		currentWidth := 4 // base padding
		requestWidth := 20
		numCols := 0
		for _, col := range columnsToConsider {
			if col.ID == ColRequest {
				continue
			}
			currentWidth += col.Width + SeparatorWidth
			numCols++
		}
		currentWidth += requestWidth + SeparatorWidth

		if currentWidth <= width || numCols <= 3 {
			// We fit or can't remove more
			break
		}

		// Find highest priority column to remove (highest number = least important)
		maxPriority := -1
		removeIdx := -1
		for i, col := range columnsToConsider {
			if col.ID == ColRequest || col.ID == ColNum || col.ID == ColName || col.ID == ColStatus {
				continue // Never remove these
			}
			if col.Priority > maxPriority {
				maxPriority = col.Priority
				removeIdx = i
			}
		}

		if removeIdx == -1 {
			break // Nothing more to remove
		}

		// Remove the column
		columnsToConsider = append(columnsToConsider[:removeIdx], columnsToConsider[removeIdx+1:]...)
	}

	// Calculate final request width
	usedWidth := 4
	for _, col := range columnsToConsider {
		if col.ID != ColRequest {
			usedWidth += col.Width + SeparatorWidth
		}
	}
	remaining := width - usedWidth - SeparatorWidth
	if remaining < 15 {
		remaining = 15
	}

	// Build final list
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
