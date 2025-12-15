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

// EventHistory displays workflow event history with a side panel for details.
type EventHistory struct {
	*tview.Flex
	app         *App
	workflowID  string
	runID       string
	table       *ui.Table
	leftPanel   *ui.Panel
	rightPanel  *ui.Panel
	sidePanel   *tview.TextView
	events      []temporal.HistoryEvent
	sidePanelOn bool
	loading     bool
}

// NewEventHistory creates a new event history view.
func NewEventHistory(app *App, workflowID, runID string) *EventHistory {
	eh := &EventHistory{
		Flex:        tview.NewFlex().SetDirection(tview.FlexColumn),
		app:         app,
		workflowID:  workflowID,
		runID:       runID,
		table:       ui.NewTable(),
		sidePanel:   tview.NewTextView(),
		events:      []temporal.HistoryEvent{},
		sidePanelOn: true,
	}
	eh.setup()
	return eh
}

func (eh *EventHistory) setup() {
	eh.SetBackgroundColor(ui.ColorBg)

	eh.table.SetHeaders("ID", "TIME", "TYPE", "DETAILS")
	eh.table.SetBorder(false)
	eh.table.SetBackgroundColor(ui.ColorBg)

	// Configure side panel
	eh.sidePanel.SetDynamicColors(true)
	eh.sidePanel.SetTextAlign(tview.AlignLeft)
	eh.sidePanel.SetBackgroundColor(ui.ColorBg)

	// Create panels
	eh.leftPanel = ui.NewPanel("Events")
	eh.leftPanel.SetContent(eh.table)

	eh.rightPanel = ui.NewPanel("Details")
	eh.rightPanel.SetContent(eh.sidePanel)

	// Selection change handler
	eh.table.SetSelectionChangedFunc(func(row, col int) {
		if eh.sidePanelOn && row > 0 {
			eh.updateSidePanel(row - 1)
		}
	})

	// Selection handler (Enter key)
	eh.table.SetSelectedFunc(func(row, col int) {
		if row > 0 {
			eh.toggleSidePanel()
			if eh.sidePanelOn {
				eh.updateSidePanel(row - 1)
			}
		}
	})

	eh.buildLayout()
}

func (eh *EventHistory) buildLayout() {
	eh.Clear()
	if eh.sidePanelOn {
		eh.AddItem(eh.leftPanel, 0, 3, true)
		eh.AddItem(eh.rightPanel, 0, 2, false)
	} else {
		eh.AddItem(eh.leftPanel, 0, 1, true)
	}
}

func (eh *EventHistory) setLoading(loading bool) {
	eh.loading = loading
}

func (eh *EventHistory) loadData() {
	provider := eh.app.Provider()
	if provider == nil {
		// Fallback to mock data if no provider
		eh.loadMockData()
		return
	}

	eh.setLoading(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Longer timeout for history
		defer cancel()

		events, err := provider.GetWorkflowHistory(ctx, eh.app.CurrentNamespace(), eh.workflowID, eh.runID)

		eh.app.UI().QueueUpdateDraw(func() {
			eh.setLoading(false)
			if err != nil {
				eh.showError(err)
				return
			}
			eh.events = events
			eh.populateTable()
		})
	}()
}

func (eh *EventHistory) loadMockData() {
	// Mock data fallback when no provider is configured
	now := time.Now()
	eh.events = []temporal.HistoryEvent{
		{ID: 1, Type: "WorkflowExecutionStarted", Time: now.Add(-5 * time.Minute), Details: "WorkflowType: MockWorkflow, TaskQueue: mock-tasks"},
		{ID: 2, Type: "WorkflowTaskScheduled", Time: now.Add(-5 * time.Minute), Details: "TaskQueue: mock-tasks"},
		{ID: 3, Type: "WorkflowTaskStarted", Time: now.Add(-5 * time.Minute), Details: "Identity: worker-1@host"},
		{ID: 4, Type: "WorkflowTaskCompleted", Time: now.Add(-5 * time.Minute), Details: "ScheduledEventId: 2"},
		{ID: 5, Type: "ActivityTaskScheduled", Time: now.Add(-4 * time.Minute), Details: "ActivityType: MockActivity, TaskQueue: mock-tasks"},
		{ID: 6, Type: "ActivityTaskStarted", Time: now.Add(-4 * time.Minute), Details: "Identity: worker-1@host, Attempt: 1"},
		{ID: 7, Type: "ActivityTaskCompleted", Time: now.Add(-3 * time.Minute), Details: "ScheduledEventId: 5, Result: {success: true}"},
	}
	eh.populateTable()
}

func (eh *EventHistory) populateTable() {
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
		eh.table.SelectRow(0)
		if len(eh.events) > 0 {
			eh.updateSidePanel(0)
		}
	}
}

func (eh *EventHistory) showError(err error) {
	eh.table.ClearRows()
	eh.table.SetHeaders("ID", "TIME", "TYPE", "DETAILS")
	eh.table.AddColoredRow(ui.ColorFailed,
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

func (eh *EventHistory) updateSidePanel(index int) {
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
		ui.TagPanelTitle,
		ui.TagFg, ev.ID,
		ui.TagPanelTitle,
		colorTag, icon, ev.Type,
		ui.TagPanelTitle,
		ui.TagFg, ev.Time.Format("2006-01-02 15:04:05.000"),
		ui.TagPanelTitle,
		ui.TagFgDim, ev.Details,
	)
	eh.sidePanel.SetText(text)
}

// Name returns the view name.
func (eh *EventHistory) Name() string {
	return "events"
}

// Start is called when the view becomes active.
func (eh *EventHistory) Start() {
	eh.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'p':
			eh.toggleSidePanel()
			if eh.sidePanelOn {
				row := eh.table.SelectedRow()
				if row >= 0 {
					eh.updateSidePanel(row)
				}
			}
			return nil
		case 'r':
			eh.loadData()
			return nil
		}
		return event
	})
	// Load data when view becomes active
	eh.loadData()
}

// Stop is called when the view is deactivated.
func (eh *EventHistory) Stop() {
	eh.table.SetInputCapture(nil)
}

// Hints returns keybinding hints for this view.
func (eh *EventHistory) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "enter", Description: "Toggle Detail"},
		{Key: "p", Description: "Preview"},
		{Key: "r", Description: "Refresh"},
		{Key: "j/k", Description: "Navigate"},
		{Key: "esc", Description: "Back"},
	}
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
		return ui.ColorRunning
	case contains(eventType, "Completed"):
		return ui.ColorCompleted
	case contains(eventType, "Failed"):
		return ui.ColorFailed
	case contains(eventType, "Scheduled"):
		return ui.ColorFgDim
	default:
		return ui.ColorFg
	}
}

// eventColorTag returns a color tag for the event type.
func eventColorTag(eventType string) string {
	switch {
	case contains(eventType, "Started"):
		return ui.TagRunning
	case contains(eventType, "Completed"):
		return ui.TagCompleted
	case contains(eventType, "Failed"):
		return ui.TagFailed
	default:
		return ui.TagFg
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
