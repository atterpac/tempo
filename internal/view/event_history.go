package view

import (
	"context"
	"fmt"
	"time"

	"github.com/atterpac/loom/internal/config"
	"github.com/atterpac/loom/internal/temporal"
	"github.com/atterpac/loom/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// EventViewMode represents the display mode for event history.
type EventViewMode int

const (
	ViewModeList EventViewMode = iota
	ViewModeTree
	ViewModeTimeline
)

// EventHistory displays workflow event history with multiple view modes.
type EventHistory struct {
	*tview.Flex
	app        *App
	workflowID string
	runID      string

	// View mode
	viewMode EventViewMode

	// List view components (original)
	table *ui.Table

	// Tree view components
	treeView  *ui.EventTreeView
	treeNodes []*temporal.EventTreeNode

	// Timeline view components
	timelineView *ui.TimelineView

	// Shared components
	leftPanel   *ui.Panel
	rightPanel  *ui.Panel
	sidePanel   *tview.TextView
	sidePanelOn bool

	// Data
	events           []temporal.HistoryEvent
	enhancedEvents   []temporal.EnhancedHistoryEvent
	loading          bool
	unsubscribeTheme func()
}

// NewEventHistory creates a new event history view.
func NewEventHistory(app *App, workflowID, runID string) *EventHistory {
	eh := &EventHistory{
		Flex:        tview.NewFlex().SetDirection(tview.FlexColumn),
		app:         app,
		workflowID:  workflowID,
		runID:       runID,
		viewMode:    ViewModeTree, // Default to tree view
		table:       ui.NewTable(),
		treeView:    ui.NewEventTreeView(),
		timelineView: ui.NewTimelineView(),
		sidePanel:   tview.NewTextView(),
		sidePanelOn: true,
	}
	eh.setup()
	return eh
}

func (eh *EventHistory) setup() {
	eh.SetBackgroundColor(ui.ColorBg())

	// Configure list view table
	eh.table.SetHeaders("ID", "TIME", "TYPE", "DETAILS")
	eh.table.SetBorder(false)
	eh.table.SetBackgroundColor(ui.ColorBg())

	// Configure side panel
	eh.sidePanel.SetDynamicColors(true)
	eh.sidePanel.SetTextAlign(tview.AlignLeft)
	eh.sidePanel.SetBackgroundColor(ui.ColorBg())

	// Create panels
	eh.leftPanel = ui.NewPanel("Events (Tree)")
	eh.rightPanel = ui.NewPanel("Details")
	eh.rightPanel.SetContent(eh.sidePanel)

	// List view selection handlers
	eh.table.SetSelectionChangedFunc(func(row, col int) {
		if eh.viewMode == ViewModeList && eh.sidePanelOn && row > 0 {
			eh.updateSidePanelFromList(row - 1)
		}
	})

	eh.table.SetSelectedFunc(func(row, col int) {
		if row > 0 {
			eh.toggleSidePanel()
			if eh.sidePanelOn {
				eh.updateSidePanelFromList(row - 1)
			}
		}
	})

	// Tree view selection handlers
	eh.treeView.SetOnSelectionChanged(func(node *temporal.EventTreeNode) {
		if eh.viewMode == ViewModeTree && eh.sidePanelOn {
			eh.updateSidePanelFromTree(node)
		}
	})

	eh.treeView.SetOnSelect(func(node *temporal.EventTreeNode) {
		// Toggle expand/collapse is handled by tree view itself
		// Optionally toggle side panel on enter
	})

	// Timeline view selection handler
	eh.timelineView.SetOnSelect(func(lane *ui.TimelineLane) {
		if lane != nil && lane.Node != nil {
			eh.updateSidePanelFromTree(lane.Node)
		}
	})

	// Register for theme changes
	eh.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		eh.SetBackgroundColor(ui.ColorBg())
		eh.sidePanel.SetBackgroundColor(ui.ColorBg())
		eh.refreshCurrentView()
	})

	eh.buildLayout()
}

func (eh *EventHistory) buildLayout() {
	eh.Clear()

	// Update panel title and content based on view mode
	switch eh.viewMode {
	case ViewModeList:
		eh.leftPanel.SetTitle("Events (List)")
		eh.leftPanel.SetContent(eh.table)
	case ViewModeTree:
		eh.leftPanel.SetTitle("Events (Tree)")
		eh.leftPanel.SetContent(eh.treeView)
	case ViewModeTimeline:
		eh.leftPanel.SetTitle("Events (Timeline)")
		eh.leftPanel.SetContent(eh.timelineView)
	}

	if eh.sidePanelOn {
		eh.AddItem(eh.leftPanel, 0, 3, true)
		eh.AddItem(eh.rightPanel, 0, 2, false)
	} else {
		eh.AddItem(eh.leftPanel, 0, 1, true)
	}

	// Set focus to the active view component
	if eh.app != nil && eh.app.UI() != nil {
		switch eh.viewMode {
		case ViewModeList:
			eh.app.UI().SetFocus(eh.table)
		case ViewModeTree:
			eh.app.UI().SetFocus(eh.treeView)
		case ViewModeTimeline:
			eh.app.UI().SetFocus(eh.timelineView)
		}
	}
}

func (eh *EventHistory) setViewMode(mode EventViewMode) {
	if eh.viewMode == mode {
		return
	}
	eh.viewMode = mode
	eh.buildLayout()
	eh.setupInputCapture()
	eh.refreshCurrentView()
}

func (eh *EventHistory) cycleViewMode() {
	nextMode := (eh.viewMode + 1) % 3
	eh.setViewMode(nextMode)
}

func (eh *EventHistory) refreshCurrentView() {
	switch eh.viewMode {
	case ViewModeList:
		eh.populateTable()
	case ViewModeTree:
		eh.populateTreeView()
	case ViewModeTimeline:
		eh.populateTimelineView()
	}
}

func (eh *EventHistory) setLoading(loading bool) {
	eh.loading = loading
}

func (eh *EventHistory) loadData() {
	provider := eh.app.Provider()
	if provider == nil {
		eh.loadMockData()
		return
	}

	eh.setLoading(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Load enhanced events for tree/timeline views
		enhancedEvents, err := provider.GetEnhancedWorkflowHistory(ctx, eh.app.CurrentNamespace(), eh.workflowID, eh.runID)

		eh.app.UI().QueueUpdateDraw(func() {
			eh.setLoading(false)
			if err != nil {
				eh.showError(err)
				return
			}

			eh.enhancedEvents = enhancedEvents

			// Convert to basic events for list view
			eh.events = make([]temporal.HistoryEvent, len(enhancedEvents))
			for i, ev := range enhancedEvents {
				eh.events[i] = temporal.HistoryEvent{
					ID:      ev.ID,
					Type:    ev.Type,
					Time:    ev.Time,
					Details: ev.Details,
				}
			}

			// Build tree nodes
			eh.treeNodes = temporal.BuildEventTree(enhancedEvents)

			// Populate current view
			eh.refreshCurrentView()
		})
	}()
}

func (eh *EventHistory) loadMockData() {
	now := time.Now()

	// Create mock enhanced events
	eh.enhancedEvents = []temporal.EnhancedHistoryEvent{
		{ID: 1, Type: "WorkflowExecutionStarted", Time: now.Add(-5 * time.Minute), Details: "WorkflowType: MockWorkflow, TaskQueue: mock-tasks", TaskQueue: "mock-tasks"},
		{ID: 2, Type: "WorkflowTaskScheduled", Time: now.Add(-5 * time.Minute), Details: "TaskQueue: mock-tasks", TaskQueue: "mock-tasks"},
		{ID: 3, Type: "WorkflowTaskStarted", Time: now.Add(-5 * time.Minute), Details: "Identity: worker-1@host", ScheduledEventID: 2, Identity: "worker-1@host"},
		{ID: 4, Type: "WorkflowTaskCompleted", Time: now.Add(-5 * time.Minute), Details: "ScheduledEventId: 2", ScheduledEventID: 2, StartedEventID: 3},
		{ID: 5, Type: "ActivityTaskScheduled", Time: now.Add(-4 * time.Minute), Details: "ActivityType: ValidateOrder, TaskQueue: mock-tasks", ActivityType: "ValidateOrder", ActivityID: "1", TaskQueue: "mock-tasks"},
		{ID: 6, Type: "ActivityTaskStarted", Time: now.Add(-4 * time.Minute), Details: "Identity: worker-1@host, Attempt: 1", ScheduledEventID: 5, Attempt: 1, Identity: "worker-1@host"},
		{ID: 7, Type: "ActivityTaskCompleted", Time: now.Add(-3 * time.Minute), Details: "ScheduledEventId: 5, Result: {success: true}", ScheduledEventID: 5, StartedEventID: 6, Result: "{success: true}"},
		{ID: 8, Type: "ActivityTaskScheduled", Time: now.Add(-3 * time.Minute), Details: "ActivityType: ProcessPayment, TaskQueue: mock-tasks", ActivityType: "ProcessPayment", ActivityID: "2", TaskQueue: "mock-tasks"},
		{ID: 9, Type: "ActivityTaskStarted", Time: now.Add(-3 * time.Minute), Details: "Identity: worker-1@host, Attempt: 1", ScheduledEventID: 8, Attempt: 1, Identity: "worker-1@host"},
		{ID: 10, Type: "ActivityTaskFailed", Time: now.Add(-2 * time.Minute), Details: "ScheduledEventId: 8, Failure: timeout", ScheduledEventID: 8, StartedEventID: 9, Failure: "timeout"},
		{ID: 11, Type: "ActivityTaskStarted", Time: now.Add(-2 * time.Minute), Details: "Identity: worker-1@host, Attempt: 2", ScheduledEventID: 8, Attempt: 2, Identity: "worker-1@host"},
		{ID: 12, Type: "ActivityTaskCompleted", Time: now.Add(-1 * time.Minute), Details: "ScheduledEventId: 8, Result: {paid: true}", ScheduledEventID: 8, StartedEventID: 11, Result: "{paid: true}"},
		{ID: 13, Type: "TimerStarted", Time: now.Add(-1 * time.Minute), Details: "TimerId: wait-30s", TimerID: "wait-30s"},
		{ID: 14, Type: "TimerFired", Time: now.Add(-30 * time.Second), Details: "TimerId: wait-30s, StartedEventId: 13", TimerID: "wait-30s", StartedEventID: 13},
	}

	// Convert to basic events
	eh.events = make([]temporal.HistoryEvent, len(eh.enhancedEvents))
	for i, ev := range eh.enhancedEvents {
		eh.events[i] = temporal.HistoryEvent{
			ID:      ev.ID,
			Type:    ev.Type,
			Time:    ev.Time,
			Details: ev.Details,
		}
	}

	// Build tree nodes
	eh.treeNodes = temporal.BuildEventTree(eh.enhancedEvents)

	// Populate current view
	eh.refreshCurrentView()
}

func (eh *EventHistory) populateTable() {
	// Preserve current selection
	currentRow := eh.table.SelectedRow()

	eh.table.ClearRows()
	eh.table.SetHeaders("ID", "TIME", "TYPE", "DETAILS")

	for _, ev := range eh.events {
		icon := eventIcon(ev.Type)
		color := eventColor(ev.Type)
		eh.table.AddColoredRow(color,
			fmt.Sprintf("%d", ev.ID),
			ev.Time.Format("15:04:05"),
			icon+" "+ev.Type,
			truncate(ev.Details, 40),
		)
	}

	if eh.table.RowCount() > 0 {
		// Restore previous selection if valid, otherwise select first row
		if currentRow >= 0 && currentRow < len(eh.events) {
			eh.table.SelectRow(currentRow)
			eh.updateSidePanelFromList(currentRow)
		} else {
			eh.table.SelectRow(0)
			if len(eh.events) > 0 {
				eh.updateSidePanelFromList(0)
			}
		}
	}
}

func (eh *EventHistory) populateTreeView() {
	eh.treeView.SetNodes(eh.treeNodes)
	if len(eh.treeNodes) > 0 {
		eh.updateSidePanelFromTree(eh.treeNodes[0])
	}
}

func (eh *EventHistory) populateTimelineView() {
	eh.timelineView.SetNodes(eh.treeNodes)
}

func (eh *EventHistory) showError(err error) {
	eh.table.ClearRows()
	eh.table.SetHeaders("ID", "TIME", "TYPE", "DETAILS")
	eh.table.AddColoredRow(ui.ColorFailed(),
		"",
		"",
		ui.IconFailed+" Error loading events",
		err.Error(),
	)
}

func (eh *EventHistory) toggleSidePanel() {
	eh.sidePanelOn = !eh.sidePanelOn
	eh.buildLayout()
}

func (eh *EventHistory) updateSidePanelFromList(index int) {
	if index < 0 || index >= len(eh.events) {
		return
	}

	ev := eh.events[index]
	icon := eventIcon(ev.Type)
	colorTag := eventColorTag(ev.Type)

	text := fmt.Sprintf(`
[%s::b]Event ID[-:-:-]
[%s]%d[-]

[%s::b]Type[-:-:-]
[%s]%s %s[-]

[%s::b]Time[-:-:-]
[%s]%s[-]

[%s::b]Details[-:-:-]
[%s]%s[-]`,
		ui.TagPanelTitle(),
		ui.TagFg(), ev.ID,
		ui.TagPanelTitle(),
		colorTag, icon, ev.Type,
		ui.TagPanelTitle(),
		ui.TagFg(), ev.Time.Format("2006-01-02 15:04:05.000"),
		ui.TagPanelTitle(),
		ui.TagFgDim(), ev.Details,
	)
	eh.sidePanel.SetText(text)
}

func (eh *EventHistory) updateSidePanelFromTree(node *temporal.EventTreeNode) {
	if node == nil {
		return
	}

	statusTag := ui.StatusColorTag(node.Status)
	icon := ui.StatusIcon(node.Status)

	var durationStr string
	if node.Duration > 0 {
		durationStr = temporal.FormatDuration(node.Duration)
	} else {
		durationStr = "running..."
	}

	var attemptsStr string
	if node.Attempts > 1 {
		attemptsStr = fmt.Sprintf("\n\n[%s::b]Attempts[-:-:-]\n[%s]%d[-]", ui.TagPanelTitle(), ui.TagFg(), node.Attempts)
	}

	var eventsStr string
	if len(node.Events) > 0 {
		eventsStr = fmt.Sprintf("\n\n[%s::b]Events[-:-:-]", ui.TagPanelTitle())
		for _, ev := range node.Events {
			evIcon := eventIcon(ev.Type)
			eventsStr += fmt.Sprintf("\n[%s]%s %s[-] [%s](%d)[-]",
				eventColorTag(ev.Type), evIcon, ev.Type, ui.TagFgDim(), ev.ID)
		}
	}

	text := fmt.Sprintf(`
[%s::b]Name[-:-:-]
[%s]%s[-]

[%s::b]Status[-:-:-]
[%s]%s %s[-]

[%s::b]Duration[-:-:-]
[%s]%s[-]

[%s::b]Start Time[-:-:-]
[%s]%s[-]%s%s`,
		ui.TagPanelTitle(),
		ui.TagFg(), node.Name,
		ui.TagPanelTitle(),
		statusTag, icon, node.Status,
		ui.TagPanelTitle(),
		ui.TagFg(), durationStr,
		ui.TagPanelTitle(),
		ui.TagFg(), node.StartTime.Format("2006-01-02 15:04:05.000"),
		attemptsStr,
		eventsStr,
	)
	eh.sidePanel.SetText(text)
}

// Name returns the view name.
func (eh *EventHistory) Name() string {
	return "events"
}

// Start is called when the view becomes active.
func (eh *EventHistory) Start() {
	// Set up input capture for the current view mode
	eh.setupInputCapture()
	// Load data when view becomes active
	eh.loadData()
}

func (eh *EventHistory) setupInputCapture() {
	// Clear all input captures first
	eh.table.SetInputCapture(nil)
	eh.treeView.SetInputCapture(nil)
	eh.timelineView.SetInputCapture(nil)

	// Common input handler for all modes
	inputHandler := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'v':
			eh.cycleViewMode()
			return nil
		case '1':
			eh.setViewMode(ViewModeList)
			return nil
		case '2':
			eh.setViewMode(ViewModeTree)
			return nil
		case '3':
			eh.setViewMode(ViewModeTimeline)
			return nil
		case 'p':
			eh.toggleSidePanel()
			return nil
		case 'r':
			eh.loadData()
			return nil
		}

		// View-specific handlers
		switch eh.viewMode {
		case ViewModeTree:
			switch event.Rune() {
			case 'e':
				eh.treeView.ExpandAll()
				return nil
			case 'c':
				eh.treeView.CollapseAll()
				return nil
			case 'f':
				eh.treeView.JumpToFailed()
				return nil
			}
		case ViewModeTimeline:
			// Timeline handles its own input via InputHandler
		}

		return event
	}

	// Apply input capture to the appropriate component
	switch eh.viewMode {
	case ViewModeList:
		eh.table.SetInputCapture(inputHandler)
	case ViewModeTree:
		eh.treeView.SetInputCapture(inputHandler)
	case ViewModeTimeline:
		eh.timelineView.SetInputCapture(inputHandler)
	}
}

// Stop is called when the view is deactivated.
func (eh *EventHistory) Stop() {
	eh.table.SetInputCapture(nil)
	eh.treeView.SetInputCapture(nil)
	eh.timelineView.SetInputCapture(nil)
	if eh.unsubscribeTheme != nil {
		eh.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	eh.table.Destroy()
	eh.treeView.Destroy()
	eh.timelineView.Destroy()
	eh.leftPanel.Destroy()
	eh.rightPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (eh *EventHistory) Hints() []ui.KeyHint {
	hints := []ui.KeyHint{
		{Key: "v", Description: "Cycle View"},
		{Key: "1/2/3", Description: "List/Tree/Timeline"},
		{Key: "p", Description: "Preview"},
		{Key: "r", Description: "Refresh"},
	}

	// Add view-specific hints
	switch eh.viewMode {
	case ViewModeTree:
		hints = append(hints,
			ui.KeyHint{Key: "e", Description: "Expand All"},
			ui.KeyHint{Key: "c", Description: "Collapse All"},
			ui.KeyHint{Key: "f", Description: "Jump to Failed"},
		)
	case ViewModeTimeline:
		hints = append(hints,
			ui.KeyHint{Key: "+/-", Description: "Zoom"},
			ui.KeyHint{Key: "h/l", Description: "Scroll"},
		)
	}

	hints = append(hints,
		ui.KeyHint{Key: "j/k", Description: "Navigate"},
		ui.KeyHint{Key: "T", Description: "Theme"},
		ui.KeyHint{Key: "esc", Description: "Back"},
	)

	return hints
}

// Focus sets focus to the current view's primary component.
func (eh *EventHistory) Focus(delegate func(p tview.Primitive)) {
	switch eh.viewMode {
	case ViewModeList:
		delegate(eh.table)
	case ViewModeTree:
		delegate(eh.treeView)
	case ViewModeTimeline:
		delegate(eh.timelineView)
	default:
		delegate(eh.table)
	}
}

// Draw applies theme colors dynamically and draws the view.
func (eh *EventHistory) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	eh.SetBackgroundColor(bg)
	eh.sidePanel.SetBackgroundColor(bg)
	eh.Flex.Draw(screen)
}

// eventIcon returns an icon for the event type.
func eventIcon(eventType string) string {
	switch {
	case contains(eventType, "Started"):
		return ui.IconRunning
	case contains(eventType, "Completed"):
		return ui.IconCompleted
	case contains(eventType, "Failed"):
		return ui.IconFailed
	case contains(eventType, "Scheduled"):
		return ui.IconPending
	case contains(eventType, "Timer"):
		return ui.IconTimedOut
	case contains(eventType, "Signal"):
		return ui.IconActivity
	case contains(eventType, "Child"):
		return ui.IconWorkflow
	default:
		return ui.IconEvent
	}
}

// eventColor returns a color for the event type.
func eventColor(eventType string) tcell.Color {
	switch {
	case contains(eventType, "Started"):
		return ui.ColorRunning()
	case contains(eventType, "Completed"):
		return ui.ColorCompleted()
	case contains(eventType, "Failed"):
		return ui.ColorFailed()
	case contains(eventType, "Scheduled"):
		return ui.ColorFgDim()
	default:
		return ui.ColorFg()
	}
}

// eventColorTag returns a color tag for the event type.
func eventColorTag(eventType string) string {
	switch {
	case contains(eventType, "Started"):
		return ui.TagRunning()
	case contains(eventType, "Completed"):
		return ui.TagCompleted()
	case contains(eventType, "Failed"):
		return ui.TagFailed()
	default:
		return ui.TagFg()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
