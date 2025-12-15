package view

import (
	"context"
	"fmt"
	"time"

	"github.com/atterpac/temportui/internal/temporal"
	"github.com/atterpac/temportui/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WorkflowList displays a list of workflows with a preview panel.
type WorkflowList struct {
	*tview.Flex
	app           *App
	namespace     string
	table         *ui.Table
	leftPanel     *ui.Panel
	rightPanel    *ui.Panel
	preview       *tview.TextView
	workflows     []temporal.Workflow
	filterText    string
	loading       bool
	autoRefresh   bool
	showPreview   bool
	refreshTicker *time.Ticker
	stopRefresh   chan struct{}
}

// NewWorkflowList creates a new workflow list view.
func NewWorkflowList(app *App, namespace string) *WorkflowList {
	wl := &WorkflowList{
		Flex:        tview.NewFlex().SetDirection(tview.FlexColumn),
		app:         app,
		namespace:   namespace,
		table:       ui.NewTable(),
		preview:     tview.NewTextView(),
		workflows:   []temporal.Workflow{},
		showPreview: true,
		stopRefresh: make(chan struct{}),
	}
	wl.setup()
	return wl
}

func (wl *WorkflowList) setup() {
	wl.table.SetHeaders("WORKFLOW ID", "TYPE", "STATUS", "START TIME")
	wl.table.SetBorder(false)
	wl.table.SetBackgroundColor(ui.ColorBg)
	wl.SetBackgroundColor(ui.ColorBg)

	// Configure preview
	wl.preview.SetDynamicColors(true)
	wl.preview.SetBackgroundColor(ui.ColorBg)
	wl.preview.SetTextColor(ui.ColorFg)
	wl.preview.SetWordWrap(true)

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
		ui.TagPanelTitle,
		ui.TagFg, truncate(w.ID, 35),
		ui.TagFgDim,
		statusColor, statusIcon, w.Status,
		ui.TagFgDim,
		ui.TagFg, w.Type,
		ui.TagFgDim,
		ui.TagFg, formatRelativeTime(now, w.StartTime),
		ui.TagFgDim,
		ui.TagFg, endTimeStr,
		ui.TagFgDim,
		ui.TagFg, durationStr,
		ui.TagFgDim,
		ui.TagFg, w.TaskQueue,
		ui.TagFgDim,
		ui.TagFgDim, truncate(w.RunID, 30),
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

		opts := temporal.ListOptions{
			PageSize: 100,
			Query:    wl.filterText,
		}
		workflows, _, err := provider.ListWorkflows(ctx, wl.namespace, opts)

		wl.app.UI().QueueUpdateDraw(func() {
			wl.setLoading(false)
			if err != nil {
				wl.showError(err)
				return
			}
			wl.workflows = workflows
			wl.populateTable()
			wl.updateStats()
		})
	}()
}

func (wl *WorkflowList) loadMockData() {
	// Mock data fallback when no provider is configured
	now := time.Now()
	wl.workflows = []temporal.Workflow{
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
	wl.populateTable()
	wl.updateStats()
}

func ptr[T any](v T) *T {
	return &v
}

func (wl *WorkflowList) populateTable() {
	wl.table.ClearRows()
	wl.table.SetHeaders("WORKFLOW ID", "TYPE", "STATUS", "START TIME")

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
		wl.table.SelectRow(0)
		// Update preview for first item
		if len(wl.workflows) > 0 {
			wl.updatePreview(wl.workflows[0])
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
	wl.table.AddColoredRow(ui.ColorFailed,
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
		switch event.Rune() {
		case '/':
			wl.showFilter()
			return nil
		case 't':
			wl.app.NavigateToTaskQueues()
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
		}
		return event
	})
	// Load data when view becomes active
	wl.loadData()
}

// Stop is called when the view is deactivated.
func (wl *WorkflowList) Stop() {
	wl.table.SetInputCapture(nil)
	wl.stopAutoRefresh()
}

// Hints returns keybinding hints for this view.
func (wl *WorkflowList) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "enter", Description: "Detail"},
		{Key: "/", Description: "Filter"},
		{Key: "r", Description: "Refresh"},
		{Key: "p", Description: "Preview"},
		{Key: "a", Description: "Auto-refresh"},
		{Key: "t", Description: "Task Queues"},
		{Key: "j/k", Description: "Navigate"},
		{Key: "esc", Description: "Back"},
	}
}

func (wl *WorkflowList) showFilter() {
	// Create filter input with styling
	input := tview.NewInputField().
		SetLabel(" " + ui.IconArrowRight + " Filter: ").
		SetFieldWidth(30).
		SetFieldBackgroundColor(ui.ColorBgLight).
		SetFieldTextColor(ui.ColorFg).
		SetLabelColor(ui.ColorAccent)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			wl.filterText = input.GetText()
			wl.loadData() // Reload with filter
		}
		// Remove input and restore focus
		wl.RemoveItem(input)
		wl.app.UI().SetFocus(wl.table)
	})

	wl.AddItem(input, 1, 0, false)
	wl.app.UI().SetFocus(input)
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
