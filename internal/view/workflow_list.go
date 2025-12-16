package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atterpac/loom/internal/config"
	"github.com/atterpac/loom/internal/temporal"
	"github.com/atterpac/loom/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WorkflowList displays a list of workflows with a preview panel.
type WorkflowList struct {
	*tview.Flex
	app              *App
	namespace        string
	table            *ui.Table
	leftPanel        *ui.Panel
	rightPanel       *ui.Panel
	preview          *tview.TextView
	emptyState       *ui.EmptyState
	noResultsState   *ui.EmptyState
	allWorkflows     []temporal.Workflow // Full unfiltered list
	workflows        []temporal.Workflow // Filtered list for display
	filterText       string
	visibilityQuery  string              // Temporal visibility query
	loading          bool
	autoRefresh      bool
	showPreview      bool
	refreshTicker    *time.Ticker
	stopRefresh      chan struct{}
	selectionMode    bool   // Multi-select mode active
	searchHistory    []string // History of visibility queries
	historyIndex     int      // Current position in history (-1 = not browsing)
	maxHistorySize   int      // Maximum number of history entries
	unsubscribeTheme func()
}

// NewWorkflowList creates a new workflow list view.
func NewWorkflowList(app *App, namespace string) *WorkflowList {
	wl := &WorkflowList{
		Flex:           tview.NewFlex().SetDirection(tview.FlexColumn),
		app:            app,
		namespace:      namespace,
		table:          ui.NewTable(),
		preview:        tview.NewTextView(),
		workflows:      []temporal.Workflow{},
		showPreview:    true,
		stopRefresh:    make(chan struct{}),
		searchHistory:  make([]string, 0, 50),
		historyIndex:   -1,
		maxHistorySize: 50,
	}
	wl.setup()
	return wl
}

func (wl *WorkflowList) setup() {
	wl.table.SetHeaders("WORKFLOW ID", "TYPE", "STATUS", "START TIME")
	wl.table.SetBorder(false)
	wl.table.SetBackgroundColor(ui.ColorBg())
	wl.SetBackgroundColor(ui.ColorBg())

	// Configure preview
	wl.preview.SetDynamicColors(true)
	wl.preview.SetBackgroundColor(ui.ColorBg())
	wl.preview.SetTextColor(ui.ColorFg())
	wl.preview.SetWordWrap(true)

	// Create empty states
	wl.emptyState = ui.EmptyStateNoWorkflows()
	wl.noResultsState = ui.EmptyStateNoResults()

	// Create panels
	wl.leftPanel = ui.NewPanel("Workflows")
	wl.leftPanel.SetContent(wl.table)

	wl.rightPanel = ui.NewPanel("Preview")
	wl.rightPanel.SetContent(wl.preview)

	// Selection change handler to update preview
	wl.table.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row-1 < len(wl.workflows) {
			wl.updatePreview(wl.workflows[row-1])
		}
	})

	// Selection handler for drill-down
	wl.table.SetOnSelect(func(row int) {
		if row >= 0 && row < len(wl.workflows) {
			wf := wl.workflows[row]
			wl.app.NavigateToWorkflowDetail(wf.ID, wf.RunID)
		}
	})

	// Register for theme changes
	wl.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		wl.SetBackgroundColor(ui.ColorBg())
		wl.preview.SetBackgroundColor(ui.ColorBg())
		wl.preview.SetTextColor(ui.ColorFg())
		// Re-render table with new colors
		if len(wl.workflows) > 0 {
			wl.populateTable()
			// Explicitly update preview with new theme colors
			row := wl.table.SelectedRow()
			if row >= 0 && row < len(wl.workflows) {
				wl.updatePreview(wl.workflows[row])
			}
		}
	})

	wl.buildLayout()
}

func (wl *WorkflowList) buildLayout() {
	wl.Clear()
	if wl.showPreview {
		wl.AddItem(wl.leftPanel, 0, 3, true)
		wl.AddItem(wl.rightPanel, 0, 2, false)
	} else {
		wl.AddItem(wl.leftPanel, 0, 1, true)
	}
}

func (wl *WorkflowList) togglePreview() {
	wl.showPreview = !wl.showPreview
	wl.buildLayout()
}

func (wl *WorkflowList) updatePreview(w temporal.Workflow) {
	now := time.Now()
	statusColor := ui.StatusColorTag(w.Status)
	statusIcon := ui.StatusIcon(w.Status)

	endTimeStr := "-"
	durationStr := "-"
	if w.EndTime != nil {
		endTimeStr = formatRelativeTime(now, *w.EndTime)
		durationStr = w.EndTime.Sub(w.StartTime).Round(time.Second).String()
	} else if w.Status == "Running" {
		durationStr = time.Since(w.StartTime).Round(time.Second).String()
	}

	text := fmt.Sprintf(`[%s::b]Workflow[-:-:-]
[%s]%s[-]

[%s]Status[-]
[%s]%s %s[-]

[%s]Type[-]
[%s]%s[-]

[%s]Started[-]
[%s]%s[-]

[%s]Ended[-]
[%s]%s[-]

[%s]Duration[-]
[%s]%s[-]

[%s]Task Queue[-]
[%s]%s[-]

[%s]Run ID[-]
[%s]%s[-]`,
		ui.TagPanelTitle(),
		ui.TagFg(), truncate(w.ID, 35),
		ui.TagFgDim(),
		statusColor, statusIcon, w.Status,
		ui.TagFgDim(),
		ui.TagFg(), w.Type,
		ui.TagFgDim(),
		ui.TagFg(), formatRelativeTime(now, w.StartTime),
		ui.TagFgDim(),
		ui.TagFg(), endTimeStr,
		ui.TagFgDim(),
		ui.TagFg(), durationStr,
		ui.TagFgDim(),
		ui.TagFg(), w.TaskQueue,
		ui.TagFgDim(),
		ui.TagFgDim(), truncate(w.RunID, 30),
	)
	wl.preview.SetText(text)
}

func (wl *WorkflowList) setLoading(loading bool) {
	wl.loading = loading
}

func (wl *WorkflowList) loadData() {
	provider := wl.app.Provider()
	if provider == nil {
		// Fallback to mock data if no provider
		wl.loadMockData()
		return
	}

	wl.setLoading(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Resolve time placeholders in the query
		resolvedQuery := ui.ResolveTimePlaceholders(wl.visibilityQuery)
		opts := temporal.ListOptions{
			PageSize: 100,
			Query:    resolvedQuery, // Use visibility query if set
		}
		workflows, _, err := provider.ListWorkflows(ctx, wl.namespace, opts)

		wl.app.UI().QueueUpdateDraw(func() {
			wl.setLoading(false)
			if err != nil {
				wl.showError(err)
				return
			}
			wl.allWorkflows = workflows
			wl.applyFilter()
		})
	}()
}

// applyFilter filters allWorkflows based on filterText and updates the display.
func (wl *WorkflowList) applyFilter() {
	if wl.filterText == "" {
		wl.workflows = wl.allWorkflows
	} else {
		filter := strings.ToLower(wl.filterText)
		wl.workflows = nil
		for _, w := range wl.allWorkflows {
			// Match against workflow ID, type, or status
			if strings.Contains(strings.ToLower(w.ID), filter) ||
				strings.Contains(strings.ToLower(w.Type), filter) ||
				strings.Contains(strings.ToLower(w.Status), filter) {
				wl.workflows = append(wl.workflows, w)
			}
		}
	}
	wl.populateTable()
	wl.updateStats()
}

func (wl *WorkflowList) loadMockData() {
	// Mock data fallback when no provider is configured
	now := time.Now()
	wl.allWorkflows = []temporal.Workflow{
		{
			ID: "order-processing-abc123", RunID: "run-001-xyz", Type: "OrderWorkflow",
			Status: "Running", Namespace: wl.namespace, TaskQueue: "order-tasks",
			StartTime: now.Add(-5 * time.Minute),
		},
		{
			ID: "payment-xyz789", RunID: "run-002-abc", Type: "PaymentWorkflow",
			Status: "Completed", Namespace: wl.namespace, TaskQueue: "payment-tasks",
			StartTime: now.Add(-1 * time.Hour), EndTime: ptr(now.Add(-55 * time.Minute)),
		},
		{
			ID: "shipment-def456", RunID: "run-003-def", Type: "ShipmentWorkflow",
			Status: "Failed", Namespace: wl.namespace, TaskQueue: "shipment-tasks",
			StartTime: now.Add(-30 * time.Minute), EndTime: ptr(now.Add(-25 * time.Minute)),
		},
		{
			ID: "inventory-check-111", RunID: "run-004-ghi", Type: "InventoryWorkflow",
			Status: "Running", Namespace: wl.namespace, TaskQueue: "inventory-tasks",
			StartTime: now.Add(-10 * time.Minute),
		},
		{
			ID: "user-signup-222", RunID: "run-005-jkl", Type: "UserOnboardingWorkflow",
			Status: "Completed", Namespace: wl.namespace, TaskQueue: "user-tasks",
			StartTime: now.Add(-2 * time.Hour), EndTime: ptr(now.Add(-1*time.Hour - 45*time.Minute)),
		},
	}
	wl.applyFilter()
}

func ptr[T any](v T) *T {
	return &v
}

func (wl *WorkflowList) populateTable() {
	// Preserve current selection
	currentRow := wl.table.SelectedRow()

	wl.table.ClearRows()
	wl.table.SetHeaders("WORKFLOW ID", "TYPE", "STATUS", "START TIME")

	// Handle empty states
	if len(wl.workflows) == 0 {
		if len(wl.allWorkflows) == 0 {
			// No workflows at all
			wl.leftPanel.SetContent(wl.emptyState)
		} else {
			// No results from filter
			wl.leftPanel.SetContent(wl.noResultsState)
		}
		wl.preview.SetText("")
		return
	}

	// Show table with data
	wl.leftPanel.SetContent(wl.table)

	now := time.Now()
	for _, w := range wl.workflows {
		// Use AddStyledRow for status icon and coloring
		wl.table.AddStyledRow(w.Status,
			truncate(w.ID, 25),
			truncate(w.Type, 15),
			w.Status,
			formatRelativeTime(now, w.StartTime),
		)
	}

	if wl.table.RowCount() > 0 {
		// Restore previous selection if valid, otherwise select first row
		if currentRow >= 0 && currentRow < len(wl.workflows) {
			wl.table.SelectRow(currentRow)
			wl.updatePreview(wl.workflows[currentRow])
		} else {
			wl.table.SelectRow(0)
			if len(wl.workflows) > 0 {
				wl.updatePreview(wl.workflows[0])
			}
		}
	}
}

func (wl *WorkflowList) updateStats() {
	running, completed, failed := 0, 0, 0
	for _, w := range wl.workflows {
		switch w.Status {
		case "Running":
			running++
		case "Completed":
			completed++
		case "Failed":
			failed++
		}
	}
	wl.app.UI().StatsBar().SetWorkflowStats(running, completed, failed)
}

func (wl *WorkflowList) showError(err error) {
	wl.table.ClearRows()
	wl.table.SetHeaders("WORKFLOW ID", "TYPE", "STATUS", "START TIME")
	wl.table.AddColoredRow(ui.ColorFailed(),
		ui.IconFailed+" Error loading workflows",
		err.Error(),
		"",
		"",
	)
}

func (wl *WorkflowList) toggleAutoRefresh() {
	wl.autoRefresh = !wl.autoRefresh
	if wl.autoRefresh {
		wl.startAutoRefresh()
	} else {
		wl.stopAutoRefresh()
	}
}

func (wl *WorkflowList) startAutoRefresh() {
	wl.refreshTicker = time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-wl.refreshTicker.C:
				wl.app.UI().QueueUpdateDraw(func() {
					wl.loadData()
				})
			case <-wl.stopRefresh:
				return
			}
		}
	}()
}

func (wl *WorkflowList) stopAutoRefresh() {
	if wl.refreshTicker != nil {
		wl.refreshTicker.Stop()
		wl.refreshTicker = nil
	}
	// Signal stop to the goroutine
	select {
	case wl.stopRefresh <- struct{}{}:
	default:
	}
}

// Name returns the view name.
func (wl *WorkflowList) Name() string {
	return "workflows"
}

// Start is called when the view becomes active.
func (wl *WorkflowList) Start() {
	wl.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle space for selection toggle when in selection mode
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' && wl.selectionMode {
			wl.table.ToggleSelection()
			wl.updateSelectionPreview()
			return nil
		}

		switch event.Rune() {
		case '/':
			wl.showFilter()
			return nil
		case 'F':
			// Visibility query with autocomplete
			wl.showVisibilityQuery()
			return nil
		case 'f':
			// Query templates
			wl.showQueryTemplates()
			return nil
		case 'D':
			// Date range picker
			wl.showDateRangePicker()
			return nil
		case 't':
			wl.app.NavigateToTaskQueues()
			return nil
		case 's':
			wl.app.NavigateToSchedules()
			return nil
		case 'a':
			wl.toggleAutoRefresh()
			return nil
		case 'r':
			wl.loadData()
			return nil
		case 'p':
			wl.togglePreview()
			return nil
		case 'y':
			wl.copyWorkflowID()
			return nil
		case 'v':
			// Toggle selection mode
			wl.toggleSelectionMode()
			return nil
		case 'c':
			// Batch cancel (only in selection mode with selections)
			if wl.selectionMode && wl.table.SelectionCount() > 0 {
				wl.showBatchCancelConfirm()
				return nil
			}
		case 'X':
			// Batch terminate (only in selection mode with selections)
			if wl.selectionMode && wl.table.SelectionCount() > 0 {
				wl.showBatchTerminateConfirm()
				return nil
			}
		case 'C':
			// Clear visibility query
			if wl.visibilityQuery != "" {
				wl.clearVisibilityQuery()
				return nil
			}
		case 'L':
			// Load saved filter
			wl.showSavedFilters()
			return nil
		case 'S':
			// Save current filter
			if wl.visibilityQuery != "" {
				wl.showSaveFilter()
				return nil
			}
		case 'd':
			// Diff - open diff view with current workflow
			wl.startDiff()
			return nil
		}

		// Ctrl+A to select all in selection mode
		if event.Key() == tcell.KeyCtrlA && wl.selectionMode {
			wl.table.SelectAll()
			wl.updateSelectionPreview()
			return nil
		}

		return event
	})
	// Load data when view becomes active
	wl.loadData()
}

// Stop is called when the view is deactivated.
func (wl *WorkflowList) Stop() {
	wl.table.SetInputCapture(nil)
	wl.Flex.SetInputCapture(nil)
	wl.stopAutoRefresh()
	if wl.unsubscribeTheme != nil {
		wl.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	wl.table.Destroy()
	wl.leftPanel.Destroy()
	wl.rightPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (wl *WorkflowList) Hints() []ui.KeyHint {
	if wl.selectionMode {
		hints := []ui.KeyHint{
			{Key: "space", Description: "Select"},
			{Key: "Ctrl+A", Description: "Select All"},
			{Key: "v", Description: "Exit Select"},
		}
		if wl.table.SelectionCount() > 0 {
			hints = append(hints,
				ui.KeyHint{Key: "c", Description: "Cancel"},
				ui.KeyHint{Key: "X", Description: "Terminate"},
			)
		}
		hints = append(hints,
			ui.KeyHint{Key: "esc", Description: "Back"},
		)
		return hints
	}

	hints := []ui.KeyHint{
		{Key: "enter", Description: "Detail"},
		{Key: "/", Description: "Filter"},
		{Key: "F", Description: "Query"},
		{Key: "f", Description: "Templates"},
		{Key: "D", Description: "Date Range"},
	}
	if wl.visibilityQuery != "" {
		hints = append(hints,
			ui.KeyHint{Key: "C", Description: "Clear Query"},
			ui.KeyHint{Key: "S", Description: "Save Filter"},
		)
	}
	hints = append(hints,
		ui.KeyHint{Key: "L", Description: "Load Filter"},
		ui.KeyHint{Key: "d", Description: "Diff"},
		ui.KeyHint{Key: "v", Description: "Select Mode"},
		ui.KeyHint{Key: "y", Description: "Copy ID"},
		ui.KeyHint{Key: "r", Description: "Refresh"},
		ui.KeyHint{Key: "p", Description: "Preview"},
		ui.KeyHint{Key: "a", Description: "Auto-refresh"},
		ui.KeyHint{Key: "t", Description: "Task Queues"},
		ui.KeyHint{Key: "s", Description: "Schedules"},
		ui.KeyHint{Key: "T", Description: "Theme"},
		ui.KeyHint{Key: "?", Description: "Help"},
		ui.KeyHint{Key: "esc", Description: "Back"},
	)
	return hints
}

// Focus sets focus to the table (which has the input handlers).
func (wl *WorkflowList) Focus(delegate func(p tview.Primitive)) {
	// If showing empty state, focus the flex container instead
	if len(wl.workflows) == 0 && len(wl.allWorkflows) == 0 {
		delegate(wl.Flex)
		return
	}
	delegate(wl.table)
}

// Draw applies theme colors dynamically and draws the view.
func (wl *WorkflowList) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	wl.SetBackgroundColor(bg)
	wl.preview.SetBackgroundColor(bg)
	wl.preview.SetTextColor(ui.ColorFg())
	wl.Flex.Draw(screen)
}

func (wl *WorkflowList) showFilter() {
	// Set up command bar callbacks
	cb := wl.app.UI().CommandBar()

	// Live filtering as user types
	cb.SetOnChange(func(text string) {
		wl.filterText = text
		wl.applyFilter()
	})

	cb.SetOnSubmit(func(cmd ui.CommandType, text string) {
		wl.filterText = text
		wl.applyFilter()
	})

	cb.SetOnCancel(func() {
		wl.closeFilter()
	})

	// Show the command bar with filter mode
	wl.app.UI().ShowCommandBar(ui.CommandFilter)

	// Pre-fill with existing filter text if any
	if wl.filterText != "" {
		cb.SetText(wl.filterText)
	}
}

func (wl *WorkflowList) closeFilter() {
	wl.app.UI().HideCommandBar()
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) copyWorkflowID() {
	row := wl.table.SelectedRow()
	if row < 0 || row >= len(wl.workflows) {
		return
	}

	wf := wl.workflows[row]
	if err := ui.CopyToClipboard(wf.ID); err != nil {
		// Show error in preview
		wl.preview.SetText(fmt.Sprintf("[%s]%s Failed to copy: %s[-]",
			ui.TagFailed(), ui.IconFailed, err.Error()))
		return
	}

	// Show success feedback in preview panel
	wl.preview.SetText(fmt.Sprintf(`[%s::b]Copied to clipboard[-:-:-]

[%s]%s[-]

[%s]Workflow ID copied![-]`,
		ui.TagPanelTitle(),
		ui.TagAccent(), wf.ID,
		ui.TagCompleted()))

	// Restore preview after a brief delay
	go func() {
		time.Sleep(1500 * time.Millisecond)
		wl.app.UI().QueueUpdateDraw(func() {
			if row < len(wl.workflows) {
				wl.updatePreview(wl.workflows[row])
			}
		})
	}()
}

func formatRelativeTime(now time.Time, t time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Selection mode methods

func (wl *WorkflowList) toggleSelectionMode() {
	wl.selectionMode = !wl.selectionMode
	if wl.selectionMode {
		wl.table.EnableSelection()
		wl.leftPanel.SetTitle("Workflows (Select Mode)")
	} else {
		wl.table.DisableSelection()
		wl.leftPanel.SetTitle("Workflows")
	}
	// Update hints
	wl.app.UI().Menu().SetHints(wl.Hints())
}

func (wl *WorkflowList) updateSelectionPreview() {
	count := wl.table.SelectionCount()
	if count == 0 {
		// Show normal preview
		row := wl.table.SelectedRow()
		if row >= 0 && row < len(wl.workflows) {
			wl.updatePreview(wl.workflows[row])
		}
	} else {
		// Show selection summary
		var running, completed, failed int
		selected := wl.table.GetSelectedRows()
		for _, idx := range selected {
			if idx < len(wl.workflows) {
				switch wl.workflows[idx].Status {
				case "Running":
					running++
				case "Completed":
					completed++
				case "Failed":
					failed++
				}
			}
		}

		text := fmt.Sprintf(`[%s::b]Selected Workflows[-:-:-]
[%s]%d workflow(s)[-]

[%s]Status Breakdown[-]
[%s]%s Running: %d[-]
[%s]%s Completed: %d[-]
[%s]%s Failed: %d[-]

[%s]Press 'c' to cancel or 'X' to terminate selected workflows[-]`,
			ui.TagPanelTitle(),
			ui.TagAccent(), count,
			ui.TagFgDim(),
			ui.TagRunning(), ui.IconRunning, running,
			ui.TagCompleted(), ui.IconCompleted, completed,
			ui.TagFailed(), ui.IconFailed, failed,
			ui.TagFgDim())
		wl.preview.SetText(text)
	}
	// Update hints to reflect selection state
	wl.app.UI().Menu().SetHints(wl.Hints())
}

// Batch operation methods

func (wl *WorkflowList) showBatchCancelConfirm() {
	selected := wl.table.GetSelectedRows()
	if len(selected) == 0 {
		return
	}

	// Build batch items
	items := make([]ui.BatchItem, len(selected))
	for i, idx := range selected {
		if idx < len(wl.workflows) {
			wf := wl.workflows[idx]
			items[i] = ui.BatchItem{
				ID:     wf.ID,
				RunID:  wf.RunID,
				Status: "pending",
			}
		}
	}

	modal := ui.NewBatchConfirmModal(ui.BatchCancel, items)
	modal.SetOnConfirm(func() {
		wl.executeBatchCancel(modal, items)
	})
	modal.SetOnCancel(func() {
		wl.closeModal("batch-confirm")
	})

	wl.app.UI().Pages().AddPage("batch-confirm", modal, true, true)
	wl.app.UI().SetFocus(modal)
}

func (wl *WorkflowList) showBatchTerminateConfirm() {
	selected := wl.table.GetSelectedRows()
	if len(selected) == 0 {
		return
	}

	// Build batch items
	items := make([]ui.BatchItem, len(selected))
	for i, idx := range selected {
		if idx < len(wl.workflows) {
			wf := wl.workflows[idx]
			items[i] = ui.BatchItem{
				ID:     wf.ID,
				RunID:  wf.RunID,
				Status: "pending",
			}
		}
	}

	modal := ui.NewBatchConfirmModal(ui.BatchTerminate, items)
	modal.SetOnConfirm(func() {
		wl.executeBatchTerminate(modal, items)
	})
	modal.SetOnCancel(func() {
		wl.closeModal("batch-confirm")
	})

	wl.app.UI().Pages().AddPage("batch-confirm", modal, true, true)
	wl.app.UI().SetFocus(modal)
}

func (wl *WorkflowList) executeBatchCancel(modal *ui.BatchConfirmModal, items []ui.BatchItem) {
	provider := wl.app.Provider()
	if provider == nil {
		wl.closeModal("batch-confirm")
		return
	}

	modal.StartProgress()

	go func() {
		// Build workflow identifiers
		workflows := make([]temporal.WorkflowIdentifier, len(items))
		for i, item := range items {
			workflows[i] = temporal.WorkflowIdentifier{
				WorkflowID: item.ID,
				RunID:      item.RunID,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Execute batch cancel
		results, _ := provider.CancelWorkflows(ctx, wl.namespace, workflows)

		// Update modal with results
		for i, result := range results {
			wl.app.UI().QueueUpdateDraw(func() {
				if result.Success {
					modal.MarkItemCompleted(i)
				} else {
					modal.MarkItemFailed(i, result.Error)
				}
			})
			// Small delay for visual feedback
			time.Sleep(100 * time.Millisecond)
		}

		// After completion, refresh the workflow list
		wl.app.UI().QueueUpdateDraw(func() {
			wl.loadData()
			wl.table.ClearSelection()
		})
	}()
}

func (wl *WorkflowList) executeBatchTerminate(modal *ui.BatchConfirmModal, items []ui.BatchItem) {
	provider := wl.app.Provider()
	if provider == nil {
		wl.closeModal("batch-confirm")
		return
	}

	modal.StartProgress()

	go func() {
		// Build workflow identifiers
		workflows := make([]temporal.WorkflowIdentifier, len(items))
		for i, item := range items {
			workflows[i] = temporal.WorkflowIdentifier{
				WorkflowID: item.ID,
				RunID:      item.RunID,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Execute batch terminate
		results, _ := provider.TerminateWorkflows(ctx, wl.namespace, workflows, "Terminated via TUI batch operation")

		// Update modal with results
		for i, result := range results {
			wl.app.UI().QueueUpdateDraw(func() {
				if result.Success {
					modal.MarkItemCompleted(i)
				} else {
					modal.MarkItemFailed(i, result.Error)
				}
			})
			// Small delay for visual feedback
			time.Sleep(100 * time.Millisecond)
		}

		// After completion, refresh the workflow list
		wl.app.UI().QueueUpdateDraw(func() {
			wl.loadData()
			wl.table.ClearSelection()
		})
	}()
}

func (wl *WorkflowList) closeModal(name string) {
	wl.app.UI().Pages().RemovePage(name)
	if current := wl.app.UI().Pages().Current(); current != nil {
		wl.app.UI().SetFocus(current)
	}
}

// Visibility query methods

func (wl *WorkflowList) showVisibilityQuery() {
	autocomplete := ui.NewAutocompleteInput()

	// Pre-fill with existing query if any
	if wl.visibilityQuery != "" {
		autocomplete.SetText(wl.visibilityQuery)
	}

	// Set up history navigation
	autocomplete.SetHistoryProvider(func(direction int) string {
		if direction < 0 {
			return wl.historyPrevious()
		}
		return wl.historyNext()
	})

	autocomplete.SetOnSubmit(func(text string) {
		wl.closeVisibilityQuery()
		wl.visibilityQuery = text
		wl.addToHistory(text) // Add to history
		wl.filterText = ""    // Clear local filter when using visibility query
		wl.updatePanelTitle()
		wl.loadData() // Reload with new query
	})

	autocomplete.SetOnCancel(func() {
		wl.closeVisibilityQuery()
		wl.historyIndex = -1 // Reset history browsing
	})

	// Create a centered container for the autocomplete
	height := 12 // Base height + room for suggestions
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(autocomplete, 80, 0, true).
			AddItem(nil, 0, 1, false),
			height, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(ui.ColorBgDark())

	wl.app.UI().Pages().AddPage("visibility-query", flex, true, true)
	wl.app.UI().SetFocus(autocomplete)
}

func (wl *WorkflowList) closeVisibilityQuery() {
	wl.app.UI().Pages().RemovePage("visibility-query")
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) showQueryTemplates() {
	selector := ui.NewQueryTemplateSelector(ui.DefaultQueryTemplates)

	selector.SetOnSelect(func(template ui.QueryTemplate) {
		wl.closeQueryTemplates()

		// Check if template has placeholder
		if strings.Contains(template.Query, "${") {
			// Show input for placeholder value
			wl.showTemplateInput(template)
		} else {
			wl.visibilityQuery = template.Query
			wl.filterText = ""
			wl.updatePanelTitle()
			wl.loadData()
		}
	})

	selector.SetOnCancel(func() {
		wl.closeQueryTemplates()
	})

	// Create a centered modal for the selector
	height := len(ui.DefaultQueryTemplates) + 4
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(selector, 70, 0, true).
			AddItem(nil, 0, 1, false),
			height, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(ui.ColorBgDark())

	wl.app.UI().Pages().AddPage("query-templates", flex, true, true)
	wl.app.UI().SetFocus(selector)
}

func (wl *WorkflowList) closeQueryTemplates() {
	wl.app.UI().Pages().RemovePage("query-templates")
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) showTemplateInput(template ui.QueryTemplate) {
	// Extract placeholder name (e.g., ${type} -> type)
	query := template.Query
	placeholder := ""
	start := strings.Index(query, "${")
	end := strings.Index(query, "}")
	if start >= 0 && end > start {
		placeholder = query[start+2 : end]
	}

	modal := ui.NewInputModal("Query Value", fmt.Sprintf("Enter value for %s:", placeholder), []ui.InputField{
		{Name: "value", Label: placeholder, Placeholder: "e.g., OrderWorkflow", Required: true},
	})

	modal.SetOnSubmit(func(values map[string]string) {
		wl.closeTemplateInput()
		value := values["value"]
		// Replace placeholder in query
		wl.visibilityQuery = strings.Replace(query, "${"+placeholder+"}", "'"+value+"'", 1)
		wl.filterText = ""
		wl.updatePanelTitle()
		wl.loadData()
	})

	modal.SetOnCancel(func() {
		wl.closeTemplateInput()
	})

	wl.app.UI().Pages().AddPage("template-input", modal, true, true)
	wl.app.UI().SetFocus(modal)
}

func (wl *WorkflowList) closeTemplateInput() {
	wl.app.UI().Pages().RemovePage("template-input")
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) updatePanelTitle() {
	title := "Workflows"
	if wl.visibilityQuery != "" {
		// Show truncated query in title
		q := wl.visibilityQuery
		if len(q) > 40 {
			q = q[:37] + "..."
		}
		title = fmt.Sprintf("Workflows [%s](%s)[-]", ui.TagAccent(), q)
	} else if wl.filterText != "" {
		title = fmt.Sprintf("Workflows [%s](/%s)[-]", ui.TagFgDim(), wl.filterText)
	}
	wl.leftPanel.SetTitle(title)
}

func (wl *WorkflowList) clearVisibilityQuery() {
	wl.visibilityQuery = ""
	wl.updatePanelTitle()
	wl.loadData()
}

// Date range picker methods

func (wl *WorkflowList) showDateRangePicker() {
	picker := ui.NewDateRangePicker()

	picker.SetOnSelect(func(query string) {
		wl.closeDateRangePicker()
		if query != "" {
			// Combine with existing query or set as new
			if wl.visibilityQuery != "" && !strings.Contains(wl.visibilityQuery, "StartTime") && !strings.Contains(wl.visibilityQuery, "CloseTime") {
				wl.visibilityQuery = wl.visibilityQuery + " AND " + query
			} else {
				wl.visibilityQuery = query
			}
		} else {
			// "All time" selected - clear date-related query parts
			wl.clearDateFromQuery()
		}
		wl.filterText = ""
		wl.updatePanelTitle()
		wl.loadData()
	})

	picker.SetOnCancel(func() {
		wl.closeDateRangePicker()
	})

	// Create centered modal
	height := picker.GetHeight()
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(picker, 55, 0, true).
			AddItem(nil, 0, 1, false),
			height, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(ui.ColorBgDark())

	wl.app.UI().Pages().AddPage("date-range", flex, true, true)
	wl.app.UI().SetFocus(picker)
}

func (wl *WorkflowList) closeDateRangePicker() {
	wl.app.UI().Pages().RemovePage("date-range")
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) clearDateFromQuery() {
	// Remove StartTime and CloseTime conditions from visibility query
	// This is a simple implementation - a full parser would be more robust
	parts := strings.Split(wl.visibilityQuery, " AND ")
	var filtered []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, "StartTime") && !strings.Contains(part, "CloseTime") {
			filtered = append(filtered, part)
		}
	}
	wl.visibilityQuery = strings.Join(filtered, " AND ")
}

// Saved filter methods

func (wl *WorkflowList) showSavedFilters() {
	cfg := wl.app.Config()
	if cfg == nil {
		return
	}

	filters := cfg.GetSavedFilters()
	picker := ui.NewFilterPicker(filters, wl.visibilityQuery)

	picker.SetOnSelect(func(filter config.SavedFilter) {
		wl.closeSavedFilters()
		wl.visibilityQuery = filter.Query
		wl.filterText = ""
		wl.updatePanelTitle()
		wl.loadData()
	})

	picker.SetOnSave(func(name, query string, isDefault bool) {
		wl.closeSavedFilters()
		cfg.SaveFilter(config.SavedFilter{
			Name:      name,
			Query:     query,
			IsDefault: isDefault,
		})
		_ = cfg.Save()
	})

	picker.SetOnDelete(func(name string) {
		_ = cfg.DeleteFilter(name)
		_ = cfg.Save()
		picker.UpdateFilters(cfg.GetSavedFilters())
	})

	picker.SetOnSetDefault(func(name string) {
		_ = cfg.SetDefaultFilter(name)
		_ = cfg.Save()
		picker.UpdateFilters(cfg.GetSavedFilters())
	})

	picker.SetOnCancel(func() {
		wl.closeSavedFilters()
	})

	// Create centered modal
	height := picker.GetHeight()
	if height < 10 {
		height = 10
	}
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(picker, 70, 0, true).
			AddItem(nil, 0, 1, false),
			height, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(ui.ColorBgDark())

	wl.app.UI().Pages().AddPage("saved-filters", flex, true, true)
	wl.app.UI().SetFocus(picker)
}

func (wl *WorkflowList) closeSavedFilters() {
	wl.app.UI().Pages().RemovePage("saved-filters")
	wl.app.UI().SetFocus(wl.table)
}

func (wl *WorkflowList) showSaveFilter() {
	if wl.visibilityQuery == "" {
		return
	}

	dialog := ui.NewQuickSaveDialog(wl.visibilityQuery)

	dialog.SetOnSave(func(name string, isDefault bool) {
		wl.closeSaveFilter()
		cfg := wl.app.Config()
		if cfg != nil {
			cfg.SaveFilter(config.SavedFilter{
				Name:      name,
				Query:     wl.visibilityQuery,
				IsDefault: isDefault,
			})
			_ = cfg.Save()
		}
	})

	dialog.SetOnCancel(func() {
		wl.closeSaveFilter()
	})

	wl.app.UI().Pages().AddPage("save-filter", dialog, true, true)
	wl.app.UI().SetFocus(dialog)
}

func (wl *WorkflowList) closeSaveFilter() {
	wl.app.UI().Pages().RemovePage("save-filter")
	wl.app.UI().SetFocus(wl.table)
}

// Search history methods

// addToHistory adds a query to the search history.
func (wl *WorkflowList) addToHistory(query string) {
	if query == "" {
		return
	}

	// Don't add duplicates if same as last entry
	if len(wl.searchHistory) > 0 && wl.searchHistory[len(wl.searchHistory)-1] == query {
		return
	}

	// Remove if already exists elsewhere in history
	for i, h := range wl.searchHistory {
		if h == query {
			wl.searchHistory = append(wl.searchHistory[:i], wl.searchHistory[i+1:]...)
			break
		}
	}

	// Add to end
	wl.searchHistory = append(wl.searchHistory, query)

	// Trim if too large
	if len(wl.searchHistory) > wl.maxHistorySize {
		wl.searchHistory = wl.searchHistory[1:]
	}

	// Reset history browsing position
	wl.historyIndex = -1
}

// historyPrevious moves to the previous history entry.
func (wl *WorkflowList) historyPrevious() string {
	if len(wl.searchHistory) == 0 {
		return wl.visibilityQuery
	}

	if wl.historyIndex == -1 {
		// Start from the end
		wl.historyIndex = len(wl.searchHistory) - 1
	} else if wl.historyIndex > 0 {
		wl.historyIndex--
	}

	return wl.searchHistory[wl.historyIndex]
}

// historyNext moves to the next history entry.
func (wl *WorkflowList) historyNext() string {
	if len(wl.searchHistory) == 0 || wl.historyIndex == -1 {
		return wl.visibilityQuery
	}

	if wl.historyIndex < len(wl.searchHistory)-1 {
		wl.historyIndex++
		return wl.searchHistory[wl.historyIndex]
	}

	// Past the end - return to current query
	wl.historyIndex = -1
	return ""
}

// getHistoryStatus returns a string describing the current history position.
func (wl *WorkflowList) getHistoryStatus() string {
	if len(wl.searchHistory) == 0 {
		return ""
	}
	if wl.historyIndex == -1 {
		return fmt.Sprintf("(%d saved)", len(wl.searchHistory))
	}
	return fmt.Sprintf("(%d/%d)", wl.historyIndex+1, len(wl.searchHistory))
}

// Diff methods

func (wl *WorkflowList) startDiff() {
	row := wl.table.SelectedRow()
	if row < 0 || row >= len(wl.workflows) {
		// No workflow selected, open empty diff view
		wl.app.NavigateToWorkflowDiffEmpty()
		return
	}

	wf := wl.workflows[row]
	wl.app.NavigateToWorkflowDiff(&wf, nil)
}
