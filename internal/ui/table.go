package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Table is a generic table component with selection support.
type Table struct {
	*tview.Table
	headers []string
	actions *ActionRegistry
	onSelect func(row int)
	// Multi-select support
	selectionEnabled  bool
	selectedRows      map[int]bool // row index (0-based, excluding header) -> selected
	onSelectionChange func(selected []int)
}

// NewTable creates a new table component.
func NewTable() *Table {
	t := &Table{
		Table:   tview.NewTable(),
		actions: NewActionRegistry(),
	}
	t.SetSelectable(true, false)
	t.SetBorders(false)
	t.SetFixed(1, 0) // Fixed header row
	// Use ColorDefault to pick up tview.Styles on each draw
	t.SetBackgroundColor(tcell.ColorDefault)
	return t
}

// Destroy is a no-op kept for backward compatibility.
func (t *Table) Destroy() {}

// Draw overrides to apply current theme colors dynamically.
func (t *Table) Draw(screen tcell.Screen) {
	// Update colors from current theme before drawing
	t.SetBackgroundColor(ColorBg())
	t.SetBorderColor(ColorBorder())
	t.SetBordersColor(ColorBorder())
	t.SetSelectedStyle(tcell.StyleDefault.
		Foreground(ColorFg()).
		Background(ColorHighlight()).
		Bold(true))

	// Update cell colors
	t.refreshCellColors()

	t.Table.Draw(screen)
}

// refreshCellColors updates cell colors to match current theme.
func (t *Table) refreshCellColors() {
	rowCount := t.GetRowCount()
	colCount := t.GetColumnCount()
	bgColor := ColorBg()
	fgColor := ColorFg()
	fgDimColor := ColorFgDim()

	// Status strings to detect and update status column colors
	statusStrings := []string{"Running", "Completed", "Failed", "Canceled", "Terminated", "TimedOut", "Active", "Deprecated"}

	for row := 0; row < rowCount; row++ {
		for col := 0; col < colCount; col++ {
			cell := t.GetCell(row, col)
			if cell == nil {
				continue
			}

			// Update background for all cells
			cell.SetBackgroundColor(bgColor)

			// Header row uses dim color
			if row == 0 {
				cell.SetTextColor(fgDimColor)
				continue
			}

			// Check if this is a status cell and update its color
			text := cell.Text
			isStatusCell := false
			for _, status := range statusStrings {
				if strings.Contains(text, status) {
					cell.SetTextColor(StatusColorTcell(status))
					isStatusCell = true
					break
				}
			}

			// Non-status cells get the base foreground color
			if !isStatusCell {
				cell.SetTextColor(fgColor)
			}
		}
	}
}

// SetHeaders sets the table column headers.
func (t *Table) SetHeaders(headers ...string) {
	t.headers = headers
	for i, h := range headers {
		cell := tview.NewTableCell(" " + strings.ToLower(h)).
			SetTextColor(ColorFgDim()).
			SetBackgroundColor(ColorBg()).
			SetSelectable(false).
			SetExpansion(1)
		t.SetCell(0, i, cell)
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) int {
	row := t.GetRowCount()
	for i, v := range values {
		cell := tview.NewTableCell(" " + v).
			SetTextColor(ColorFg()).
			SetBackgroundColor(ColorBg()).
			SetExpansion(1)
		t.SetCell(row, i, cell)
	}
	return row
}

// AddColoredRow adds a row with a specific color.
func (t *Table) AddColoredRow(color tcell.Color, values ...string) int {
	row := t.GetRowCount()
	for i, v := range values {
		cell := tview.NewTableCell(" " + v).
			SetTextColor(color).
			SetBackgroundColor(ColorBg()).
			SetExpansion(1)
		t.SetCell(row, i, cell)
	}
	return row
}

// AddStyledRow adds a row with status icon and color.
func (t *Table) AddStyledRow(status string, values ...string) int {
	row := t.GetRowCount()
	color := StatusColorTcell(status)
	icon := StatusIcon(status)

	for i, v := range values {
		displayValue := " " + v
		cellColor := color

		// Add status icon to the status column (usually column 2 or 3)
		if v == status {
			displayValue = " " + icon + " " + v
		} else {
			cellColor = ColorFg()
		}

		cell := tview.NewTableCell(displayValue).
			SetTextColor(cellColor).
			SetBackgroundColor(ColorBg()).
			SetExpansion(1)
		t.SetCell(row, i, cell)
	}
	return row
}

// ClearRows removes all rows except the header.
func (t *Table) ClearRows() {
	rowCount := t.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		t.RemoveRow(i)
	}
}

// SetOnSelect sets the callback for when a row is selected.
func (t *Table) SetOnSelect(fn func(row int)) {
	t.onSelect = fn
	t.SetSelectedFunc(func(row, col int) {
		if row > 0 && fn != nil { // Skip header
			fn(row - 1) // Adjust for header
		}
	})
}

// Actions returns the action registry for this table.
func (t *Table) Actions() *ActionRegistry {
	return t.actions
}

// SelectedRow returns the currently selected row index (0-based, excluding header).
func (t *Table) SelectedRow() int {
	row, _ := t.GetSelection()
	return row - 1 // Adjust for header
}

// SelectRow selects a specific row (0-based, excluding header).
func (t *Table) SelectRow(row int) {
	t.Select(row+1, 0) // Adjust for header
}

// RowCount returns the number of data rows (excluding header).
func (t *Table) RowCount() int {
	count := t.GetRowCount()
	if count > 0 {
		return count - 1 // Exclude header
	}
	return 0
}

// StatusColor returns the appropriate color for a workflow status (deprecated, use StatusColorTcell).
func StatusColor(status string) tcell.Color {
	return StatusColorTcell(status)
}

// Multi-select methods

// EnableSelection enables multi-select mode on the table.
func (t *Table) EnableSelection() {
	t.selectionEnabled = true
	if t.selectedRows == nil {
		t.selectedRows = make(map[int]bool)
	}
}

// DisableSelection disables multi-select mode.
func (t *Table) DisableSelection() {
	t.selectionEnabled = false
	t.ClearSelection()
}

// IsSelectionEnabled returns whether multi-select is enabled.
func (t *Table) IsSelectionEnabled() bool {
	return t.selectionEnabled
}

// ToggleSelection toggles the selection state of the current row.
func (t *Table) ToggleSelection() {
	if !t.selectionEnabled {
		return
	}
	row := t.SelectedRow()
	if row < 0 {
		return
	}
	t.ToggleRowSelection(row)
}

// ToggleRowSelection toggles selection for a specific row.
func (t *Table) ToggleRowSelection(row int) {
	if !t.selectionEnabled || row < 0 {
		return
	}
	if t.selectedRows == nil {
		t.selectedRows = make(map[int]bool)
	}
	if t.selectedRows[row] {
		delete(t.selectedRows, row)
	} else {
		t.selectedRows[row] = true
	}
	t.updateRowSelectionVisual(row)
	t.notifySelectionChange()
}

// SelectRowMulti selects a row (adds to selection).
func (t *Table) SelectRowMulti(row int) {
	if !t.selectionEnabled || row < 0 {
		return
	}
	if t.selectedRows == nil {
		t.selectedRows = make(map[int]bool)
	}
	t.selectedRows[row] = true
	t.updateRowSelectionVisual(row)
	t.notifySelectionChange()
}

// DeselectRow removes a row from selection.
func (t *Table) DeselectRow(row int) {
	if !t.selectionEnabled || row < 0 {
		return
	}
	if t.selectedRows != nil {
		delete(t.selectedRows, row)
		t.updateRowSelectionVisual(row)
		t.notifySelectionChange()
	}
}

// IsRowSelected returns whether a specific row is selected.
func (t *Table) IsRowSelected(row int) bool {
	if t.selectedRows == nil {
		return false
	}
	return t.selectedRows[row]
}

// GetSelectedRows returns all selected row indices (sorted).
func (t *Table) GetSelectedRows() []int {
	if t.selectedRows == nil {
		return nil
	}
	var rows []int
	for row := range t.selectedRows {
		rows = append(rows, row)
	}
	// Sort rows
	for i := 0; i < len(rows)-1; i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i] > rows[j] {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	return rows
}

// SelectionCount returns the number of selected rows.
func (t *Table) SelectionCount() int {
	if t.selectedRows == nil {
		return 0
	}
	return len(t.selectedRows)
}

// ClearSelection clears all row selections.
func (t *Table) ClearSelection() {
	if t.selectedRows == nil {
		return
	}
	oldRows := t.GetSelectedRows()
	t.selectedRows = make(map[int]bool)
	for _, row := range oldRows {
		t.updateRowSelectionVisual(row)
	}
	t.notifySelectionChange()
}

// SelectAll selects all data rows.
func (t *Table) SelectAll() {
	if !t.selectionEnabled {
		return
	}
	if t.selectedRows == nil {
		t.selectedRows = make(map[int]bool)
	}
	rowCount := t.RowCount()
	for i := 0; i < rowCount; i++ {
		t.selectedRows[i] = true
		t.updateRowSelectionVisual(i)
	}
	t.notifySelectionChange()
}

// SetOnSelectionChange sets the callback for selection changes.
func (t *Table) SetOnSelectionChange(fn func(selected []int)) {
	t.onSelectionChange = fn
}

// updateRowSelectionVisual updates the visual appearance of a row based on selection state.
func (t *Table) updateRowSelectionVisual(row int) {
	if row < 0 {
		return
	}
	tableRow := row + 1 // Adjust for header
	colCount := t.GetColumnCount()
	isSelected := t.IsRowSelected(row)

	for col := 0; col < colCount; col++ {
		cell := t.GetCell(tableRow, col)
		if cell == nil {
			continue
		}

		// Get current cell text
		text := cell.Text
		if col == 0 {
			// First column shows selection marker
			if isSelected {
				// Add checkmark if not already present
				if !strings.HasPrefix(text, " "+IconCompleted) && !strings.HasPrefix(text, IconCompleted) {
					// Remove leading space and add checkmark
					text = strings.TrimPrefix(text, " ")
					text = " " + IconCompleted + " " + text
				}
			} else {
				// Remove checkmark if present
				if strings.Contains(text, IconCompleted) {
					text = strings.Replace(text, IconCompleted+" ", "", 1)
				}
			}
			cell.SetText(text)
		}

		// Add subtle background highlight for selected rows
		if isSelected {
			cell.SetBackgroundColor(ColorBgDark())
		} else {
			cell.SetBackgroundColor(ColorBg())
		}
	}
}

// notifySelectionChange calls the selection change callback.
func (t *Table) notifySelectionChange() {
	if t.onSelectionChange != nil {
		t.onSelectionChange(t.GetSelectedRows())
	}
}

// RefreshSelectionVisuals updates all row visuals based on selection state.
// Call this after populating the table if selection mode is enabled.
func (t *Table) RefreshSelectionVisuals() {
	if !t.selectionEnabled {
		return
	}
	rowCount := t.RowCount()
	for i := 0; i < rowCount; i++ {
		t.updateRowSelectionVisual(i)
	}
}

// RefreshColors is a no-op kept for backward compatibility.
// Colors are now refreshed automatically on each Draw().
func (t *Table) RefreshColors() {}
