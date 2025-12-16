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

// WorkflowDiff displays a side-by-side comparison of two workflows.
type WorkflowDiff struct {
	*tview.Flex
	app       *App
	namespace string

	// Workflow data
	workflowA *temporal.Workflow
	workflowB *temporal.Workflow
	eventsA   []temporal.HistoryEvent
	eventsB   []temporal.HistoryEvent

	// UI components
	leftPanel   *ui.Panel
	rightPanel  *ui.Panel
	leftInfo    *tview.TextView
	rightInfo   *tview.TextView
	leftEvents  *ui.Table
	rightEvents *ui.Table

	// State
	focusLeft        bool
	loading          bool
	unsubscribeTheme func()
}

// NewWorkflowDiff creates a new workflow diff view.
func NewWorkflowDiff(app *App, namespace string) *WorkflowDiff {
	wd := &WorkflowDiff{
		Flex:       tview.NewFlex().SetDirection(tview.FlexColumn),
		app:        app,
		namespace:  namespace,
		focusLeft:  true,
	}
	wd.setup()
	return wd
}

// NewWorkflowDiffWithWorkflows creates a diff view with pre-loaded workflows.
func NewWorkflowDiffWithWorkflows(app *App, namespace string, workflowA, workflowB *temporal.Workflow) *WorkflowDiff {
	wd := NewWorkflowDiff(app, namespace)
	wd.workflowA = workflowA
	wd.workflowB = workflowB
	return wd
}

func (wd *WorkflowDiff) setup() {
	wd.SetBackgroundColor(ui.ColorBg())

	// Create left side components
	wd.leftInfo = tview.NewTextView().SetDynamicColors(true)
	wd.leftInfo.SetBackgroundColor(ui.ColorBg())
	wd.leftEvents = ui.NewTable()
	wd.leftEvents.SetHeaders("EVENT", "TYPE", "TIME")

	leftContent := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(wd.leftInfo, 8, 0, false).
		AddItem(wd.leftEvents, 0, 1, true)
	leftContent.SetBackgroundColor(ui.ColorBg())

	wd.leftPanel = ui.NewPanel("Workflow A")
	wd.leftPanel.SetContent(leftContent)

	// Create right side components
	wd.rightInfo = tview.NewTextView().SetDynamicColors(true)
	wd.rightInfo.SetBackgroundColor(ui.ColorBg())
	wd.rightEvents = ui.NewTable()
	wd.rightEvents.SetHeaders("EVENT", "TYPE", "TIME")

	rightContent := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(wd.rightInfo, 8, 0, false).
		AddItem(wd.rightEvents, 0, 1, true)
	rightContent.SetBackgroundColor(ui.ColorBg())

	wd.rightPanel = ui.NewPanel("Workflow B")
	wd.rightPanel.SetContent(rightContent)

	// Build layout
	wd.AddItem(wd.leftPanel, 0, 1, true)
	wd.AddItem(wd.rightPanel, 0, 1, false)

	// Register for theme changes
	wd.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		wd.SetBackgroundColor(ui.ColorBg())
		wd.leftInfo.SetBackgroundColor(ui.ColorBg())
		wd.rightInfo.SetBackgroundColor(ui.ColorBg())
	})
}

// Name returns the view name.
func (wd *WorkflowDiff) Name() string {
	return "workflow-diff"
}

// Start is called when the view becomes active.
func (wd *WorkflowDiff) Start() {
	wd.leftEvents.SetInputCapture(wd.inputHandler)
	wd.rightEvents.SetInputCapture(wd.inputHandler)

	// Show empty state or prompt for workflows
	if wd.workflowA == nil && wd.workflowB == nil {
		wd.showEmptyState()
	} else {
		wd.loadData()
	}
}

// Stop is called when the view is deactivated.
func (wd *WorkflowDiff) Stop() {
	wd.leftEvents.SetInputCapture(nil)
	wd.rightEvents.SetInputCapture(nil)
	if wd.unsubscribeTheme != nil {
		wd.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	wd.leftEvents.Destroy()
	wd.rightEvents.Destroy()
	wd.leftPanel.Destroy()
	wd.rightPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (wd *WorkflowDiff) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "Tab", Description: "Switch Panel"},
		{Key: "a", Description: "Set Left"},
		{Key: "b", Description: "Set Right"},
		{Key: "r", Description: "Refresh"},
		{Key: "esc", Description: "Back"},
	}
}

// Focus sets focus to the current panel.
func (wd *WorkflowDiff) Focus(delegate func(p tview.Primitive)) {
	if wd.focusLeft {
		delegate(wd.leftEvents)
	} else {
		delegate(wd.rightEvents)
	}
}

// Draw applies theme colors dynamically and draws the view.
func (wd *WorkflowDiff) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	wd.SetBackgroundColor(bg)
	wd.leftInfo.SetBackgroundColor(bg)
	wd.rightInfo.SetBackgroundColor(bg)
	wd.Flex.Draw(screen)
}

func (wd *WorkflowDiff) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		wd.toggleFocus()
		return nil
	}

	switch event.Rune() {
	case 'a':
		wd.promptWorkflowInput(true)
		return nil
	case 'b':
		wd.promptWorkflowInput(false)
		return nil
	case 'r':
		wd.loadData()
		return nil
	}

	return event
}

func (wd *WorkflowDiff) toggleFocus() {
	wd.focusLeft = !wd.focusLeft
	if wd.focusLeft {
		wd.app.UI().SetFocus(wd.leftEvents)
		wd.leftPanel.SetBorderColor(ui.ColorAccent())
		wd.rightPanel.SetBorderColor(ui.ColorPanelBorder())
	} else {
		wd.app.UI().SetFocus(wd.rightEvents)
		wd.rightPanel.SetBorderColor(ui.ColorAccent())
		wd.leftPanel.SetBorderColor(ui.ColorPanelBorder())
	}
}

func (wd *WorkflowDiff) showEmptyState() {
	emptyText := fmt.Sprintf(`[%s::b]Workflow Comparison[-:-:-]

[%s]No workflows selected for comparison.[-]

[%s]Press 'a' to set the left workflow
Press 'b' to set the right workflow[-]`,
		ui.TagPanelTitle(),
		ui.TagFgDim(),
		ui.TagFg())

	wd.leftInfo.SetText(emptyText)
	wd.rightInfo.SetText("")
	wd.leftEvents.ClearRows()
	wd.rightEvents.ClearRows()
}

func (wd *WorkflowDiff) promptWorkflowInput(isLeft bool) {
	side := "Right"
	if isLeft {
		side = "Left"
	}

	modal := ui.NewInputModal(
		fmt.Sprintf("Set %s Workflow", side),
		"Enter workflow ID to compare",
		[]ui.InputField{
			{Name: "workflowId", Label: "Workflow ID", Placeholder: "workflow-id", Required: true},
			{Name: "runId", Label: "Run ID", Placeholder: "(optional)", Required: false},
		},
	)

	modal.SetOnSubmit(func(values map[string]string) {
		wd.closeModal("diff-input")
		workflowID := values["workflowId"]
		runID := values["runId"]

		if workflowID != "" {
			wd.loadWorkflow(isLeft, workflowID, runID)
		}
	})

	modal.SetOnCancel(func() {
		wd.closeModal("diff-input")
	})

	wd.app.UI().Pages().AddPage("diff-input", modal, true, true)
	wd.app.UI().SetFocus(modal)
}

func (wd *WorkflowDiff) closeModal(name string) {
	wd.app.UI().Pages().RemovePage(name)
	if wd.focusLeft {
		wd.app.UI().SetFocus(wd.leftEvents)
	} else {
		wd.app.UI().SetFocus(wd.rightEvents)
	}
}

func (wd *WorkflowDiff) loadWorkflow(isLeft bool, workflowID, runID string) {
	provider := wd.app.Provider()
	if provider == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		workflow, err := provider.GetWorkflow(ctx, wd.namespace, workflowID, runID)
		if err != nil {
			wd.app.UI().QueueUpdateDraw(func() {
				errorText := fmt.Sprintf("[%s]Error: %s[-]", ui.TagFailed(), err.Error())
				if isLeft {
					wd.leftInfo.SetText(errorText)
				} else {
					wd.rightInfo.SetText(errorText)
				}
			})
			return
		}

		events, _ := provider.GetWorkflowHistory(ctx, wd.namespace, workflow.ID, workflow.RunID)

		wd.app.UI().QueueUpdateDraw(func() {
			if isLeft {
				wd.workflowA = workflow
				wd.eventsA = events
				wd.leftPanel.SetTitle("Workflow A: " + truncate(workflow.ID, 25))
				wd.updateLeftInfo()
				wd.updateLeftEvents()
			} else {
				wd.workflowB = workflow
				wd.eventsB = events
				wd.rightPanel.SetTitle("Workflow B: " + truncate(workflow.ID, 25))
				wd.updateRightInfo()
				wd.updateRightEvents()
			}
		})
	}()
}

func (wd *WorkflowDiff) loadData() {
	if wd.workflowA != nil {
		wd.loadWorkflow(true, wd.workflowA.ID, wd.workflowA.RunID)
	}
	if wd.workflowB != nil {
		wd.loadWorkflow(false, wd.workflowB.ID, wd.workflowB.RunID)
	}
}

func (wd *WorkflowDiff) updateLeftInfo() {
	if wd.workflowA == nil {
		wd.leftInfo.SetText("")
		return
	}
	wd.leftInfo.SetText(wd.formatWorkflowInfo(wd.workflowA, len(wd.eventsA)))
}

func (wd *WorkflowDiff) updateRightInfo() {
	if wd.workflowB == nil {
		wd.rightInfo.SetText("")
		return
	}
	wd.rightInfo.SetText(wd.formatWorkflowInfo(wd.workflowB, len(wd.eventsB)))
}

func (wd *WorkflowDiff) formatWorkflowInfo(w *temporal.Workflow, eventCount int) string {
	statusColor := ui.StatusColorTag(w.Status)
	statusIcon := ui.StatusIcon(w.Status)

	duration := "-"
	if w.EndTime != nil {
		duration = w.EndTime.Sub(w.StartTime).Round(time.Second).String()
	} else if w.Status == "Running" {
		duration = time.Since(w.StartTime).Round(time.Second).String() + " (running)"
	}

	return fmt.Sprintf(`[%s]Type:[-] [%s]%s[-]
[%s]Status:[-] [%s]%s %s[-]
[%s]Started:[-] [%s]%s[-]
[%s]Duration:[-] [%s]%s[-]
[%s]Events:[-] [%s]%d[-]
[%s]Task Queue:[-] [%s]%s[-]`,
		ui.TagFgDim(), ui.TagFg(), w.Type,
		ui.TagFgDim(), statusColor, statusIcon, w.Status,
		ui.TagFgDim(), ui.TagFg(), w.StartTime.Format("2006-01-02 15:04:05"),
		ui.TagFgDim(), ui.TagFg(), duration,
		ui.TagFgDim(), ui.TagAccent(), eventCount,
		ui.TagFgDim(), ui.TagFg(), w.TaskQueue)
}

func (wd *WorkflowDiff) updateLeftEvents() {
	wd.leftEvents.ClearRows()
	for _, e := range wd.eventsA {
		wd.leftEvents.AddRow(
			fmt.Sprintf("%d", e.ID),
			e.Type,
			e.Time.Format("15:04:05"),
		)
	}
	if wd.leftEvents.RowCount() > 0 {
		wd.leftEvents.SelectRow(0)
	}
}

func (wd *WorkflowDiff) updateRightEvents() {
	wd.rightEvents.ClearRows()
	for _, e := range wd.eventsB {
		wd.rightEvents.AddRow(
			fmt.Sprintf("%d", e.ID),
			e.Type,
			e.Time.Format("15:04:05"),
		)
	}
	if wd.rightEvents.RowCount() > 0 {
		wd.rightEvents.SelectRow(0)
	}
}

// SetWorkflowA sets the left workflow for comparison.
func (wd *WorkflowDiff) SetWorkflowA(w *temporal.Workflow) {
	wd.workflowA = w
}

// SetWorkflowB sets the right workflow for comparison.
func (wd *WorkflowDiff) SetWorkflowB(w *temporal.Workflow) {
	wd.workflowB = w
}
