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

// ScheduleList displays a list of schedules with actions.
type ScheduleList struct {
	*tview.Flex
	app              *App
	namespace        string
	table            *ui.Table
	leftPanel        *ui.Panel
	rightPanel       *ui.Panel
	preview          *tview.TextView
	schedules        []temporal.Schedule
	loading          bool
	showPreview      bool
	unsubscribeTheme func()
}

// NewScheduleList creates a new schedule list view.
func NewScheduleList(app *App, namespace string) *ScheduleList {
	sl := &ScheduleList{
		Flex:        tview.NewFlex().SetDirection(tview.FlexColumn),
		app:         app,
		namespace:   namespace,
		table:       ui.NewTable(),
		preview:     tview.NewTextView(),
		schedules:   []temporal.Schedule{},
		showPreview: true,
	}
	sl.setup()
	return sl
}

func (sl *ScheduleList) setup() {
	sl.table.SetHeaders("SCHEDULE ID", "WORKFLOW TYPE", "SPEC", "STATUS", "NEXT RUN")
	sl.table.SetBorder(false)
	sl.table.SetBackgroundColor(ui.ColorBg())
	sl.SetBackgroundColor(ui.ColorBg())

	// Configure preview
	sl.preview.SetDynamicColors(true)
	sl.preview.SetBackgroundColor(ui.ColorBg())
	sl.preview.SetTextColor(ui.ColorFg())
	sl.preview.SetWordWrap(true)

	// Create panels
	sl.leftPanel = ui.NewPanel("Schedules")
	sl.leftPanel.SetContent(sl.table)

	sl.rightPanel = ui.NewPanel("Preview")
	sl.rightPanel.SetContent(sl.preview)

	// Selection change handler to update preview
	sl.table.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row-1 < len(sl.schedules) {
			sl.updatePreview(sl.schedules[row-1])
		}
	})

	// Register for theme changes
	sl.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		sl.SetBackgroundColor(ui.ColorBg())
		sl.preview.SetBackgroundColor(ui.ColorBg())
		sl.preview.SetTextColor(ui.ColorFg())
		if len(sl.schedules) > 0 {
			sl.populateTable()
		}
	})

	sl.buildLayout()
}

func (sl *ScheduleList) buildLayout() {
	sl.Clear()
	if sl.showPreview {
		sl.AddItem(sl.leftPanel, 0, 3, true)
		sl.AddItem(sl.rightPanel, 0, 2, false)
	} else {
		sl.AddItem(sl.leftPanel, 0, 1, true)
	}
}

func (sl *ScheduleList) togglePreview() {
	sl.showPreview = !sl.showPreview
	sl.buildLayout()
}

func (sl *ScheduleList) updatePreview(s temporal.Schedule) {
	pauseStatus := "Active"
	pauseColor := ui.TagCompleted()
	if s.Paused {
		pauseStatus = "Paused"
		pauseColor = ui.TagCanceled()
	}

	nextRun := "-"
	if s.NextRunTime != nil {
		nextRun = formatRelativeTime(time.Now(), *s.NextRunTime)
	}

	lastRun := "-"
	if s.LastRunTime != nil {
		lastRun = formatRelativeTime(time.Now(), *s.LastRunTime)
	}

	text := fmt.Sprintf(`[%s::b]Schedule[-:-:-]
[%s]%s[-]

[%s]Status[-]
[%s]%s[-]

[%s]Workflow Type[-]
[%s]%s[-]

[%s]Spec[-]
[%s]%s[-]

[%s]Next Run[-]
[%s]%s[-]

[%s]Last Run[-]
[%s]%s[-]

[%s]Total Actions[-]
[%s]%d[-]

[%s]Notes[-]
[%s]%s[-]`,
		ui.TagPanelTitle(),
		ui.TagFg(), s.ID,
		ui.TagFgDim(),
		pauseColor, pauseStatus,
		ui.TagFgDim(),
		ui.TagFg(), s.WorkflowType,
		ui.TagFgDim(),
		ui.TagFg(), s.Spec,
		ui.TagFgDim(),
		ui.TagFg(), nextRun,
		ui.TagFgDim(),
		ui.TagFg(), lastRun,
		ui.TagFgDim(),
		ui.TagFg(), s.TotalActions,
		ui.TagFgDim(),
		ui.TagFgDim(), s.Notes,
	)
	sl.preview.SetText(text)
}

func (sl *ScheduleList) loadData() {
	provider := sl.app.Provider()
	if provider == nil {
		sl.loadMockData()
		return
	}

	sl.loading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		schedules, _, err := provider.ListSchedules(ctx, sl.namespace, temporal.ListOptions{PageSize: 100})

		sl.app.UI().QueueUpdateDraw(func() {
			sl.loading = false
			if err != nil {
				sl.showError(err)
				return
			}
			sl.schedules = schedules
			sl.populateTable()
		})
	}()
}

func (sl *ScheduleList) loadMockData() {
	now := time.Now()
	nextRun := now.Add(5 * time.Minute)
	lastRun := now.Add(-1 * time.Hour)
	sl.schedules = []temporal.Schedule{
		{
			ID:           "daily-report",
			WorkflowType: "ReportWorkflow",
			Spec:         "0 9 * * *",
			Paused:       false,
			NextRunTime:  &nextRun,
			LastRunTime:  &lastRun,
			TotalActions: 365,
			Notes:        "Daily report generation",
		},
		{
			ID:           "hourly-cleanup",
			WorkflowType: "CleanupWorkflow",
			Spec:         "every 1h",
			Paused:       false,
			NextRunTime:  &nextRun,
			LastRunTime:  &lastRun,
			TotalActions: 2190,
			Notes:        "Hourly cleanup tasks",
		},
		{
			ID:           "weekly-backup",
			WorkflowType: "BackupWorkflow",
			Spec:         "0 0 * * 0",
			Paused:       true,
			NextRunTime:  nil,
			LastRunTime:  &lastRun,
			TotalActions: 52,
			Notes:        "Weekly backups (paused)",
		},
	}
	sl.populateTable()
}

func (sl *ScheduleList) populateTable() {
	// Preserve current selection
	currentRow := sl.table.SelectedRow()

	sl.table.ClearRows()
	sl.table.SetHeaders("SCHEDULE ID", "WORKFLOW TYPE", "SPEC", "STATUS", "NEXT RUN")

	for _, s := range sl.schedules {
		status := "Active"
		statusColor := ui.ColorCompleted()
		if s.Paused {
			status = "Paused"
			statusColor = ui.ColorCanceled()
		}

		nextRun := "-"
		if s.NextRunTime != nil {
			nextRun = formatRelativeTime(time.Now(), *s.NextRunTime)
		}

		sl.table.AddColoredRow(statusColor,
			truncate(s.ID, 20),
			truncate(s.WorkflowType, 20),
			truncate(s.Spec, 15),
			status,
			nextRun,
		)
	}

	if sl.table.RowCount() > 0 {
		// Restore previous selection if valid, otherwise select first row
		if currentRow >= 0 && currentRow < len(sl.schedules) {
			sl.table.SelectRow(currentRow)
			sl.updatePreview(sl.schedules[currentRow])
		} else {
			sl.table.SelectRow(0)
			if len(sl.schedules) > 0 {
				sl.updatePreview(sl.schedules[0])
			}
		}
	}
}

func (sl *ScheduleList) showError(err error) {
	sl.table.ClearRows()
	sl.table.SetHeaders("SCHEDULE ID", "WORKFLOW TYPE", "SPEC", "STATUS", "NEXT RUN")
	sl.table.AddColoredRow(ui.ColorFailed(),
		ui.IconFailed+" Error loading schedules",
		err.Error(),
		"",
		"",
		"",
	)
}

func (sl *ScheduleList) getSelectedSchedule() *temporal.Schedule {
	row := sl.table.SelectedRow() // Use SelectedRow() which accounts for header
	if row >= 0 && row < len(sl.schedules) {
		return &sl.schedules[row]
	}
	return nil
}

// Mutation methods

func (sl *ScheduleList) showPauseConfirm() {
	s := sl.getSelectedSchedule()
	if s == nil {
		return
	}

	if s.Paused {
		sl.showUnpauseConfirm(s)
		return
	}

	command := fmt.Sprintf(`temporal schedule toggle \
  --schedule-id %s \
  --namespace %s \
  --pause \
  --reason "Paused via TUI"`,
		s.ID, sl.namespace)

	modal := ui.NewConfirmModal(
		"Pause Schedule",
		fmt.Sprintf("Pause schedule %s?", s.ID),
		command,
	).SetOnConfirm(func() {
		sl.executePauseSchedule(s.ID)
	}).SetOnCancel(func() {
		sl.closeModal("confirm-pause")
	})

	sl.app.UI().Pages().AddPage("confirm-pause", modal, true, true)
	sl.app.UI().SetFocus(modal)
}

func (sl *ScheduleList) showUnpauseConfirm(s *temporal.Schedule) {
	command := fmt.Sprintf(`temporal schedule toggle \
  --schedule-id %s \
  --namespace %s \
  --unpause \
  --reason "Unpaused via TUI"`,
		s.ID, sl.namespace)

	modal := ui.NewConfirmModal(
		"Unpause Schedule",
		fmt.Sprintf("Unpause schedule %s?", s.ID),
		command,
	).SetOnConfirm(func() {
		sl.executeUnpauseSchedule(s.ID)
	}).SetOnCancel(func() {
		sl.closeModal("confirm-unpause")
	})

	sl.app.UI().Pages().AddPage("confirm-unpause", modal, true, true)
	sl.app.UI().SetFocus(modal)
}

func (sl *ScheduleList) executePauseSchedule(scheduleID string) {
	provider := sl.app.Provider()
	if provider == nil {
		sl.closeModal("confirm-pause")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.PauseSchedule(ctx, sl.namespace, scheduleID, "Paused via TUI")

		sl.app.UI().QueueUpdateDraw(func() {
			sl.closeModal("confirm-pause")
			if err != nil {
				sl.showError(err)
			} else {
				sl.loadData()
			}
		})
	}()
}

func (sl *ScheduleList) executeUnpauseSchedule(scheduleID string) {
	provider := sl.app.Provider()
	if provider == nil {
		sl.closeModal("confirm-unpause")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.UnpauseSchedule(ctx, sl.namespace, scheduleID, "Unpaused via TUI")

		sl.app.UI().QueueUpdateDraw(func() {
			sl.closeModal("confirm-unpause")
			if err != nil {
				sl.showError(err)
			} else {
				sl.loadData()
			}
		})
	}()
}

func (sl *ScheduleList) showTriggerConfirm() {
	s := sl.getSelectedSchedule()
	if s == nil {
		return
	}

	command := fmt.Sprintf(`temporal schedule trigger \
  --schedule-id %s \
  --namespace %s`,
		s.ID, sl.namespace)

	modal := ui.NewConfirmModal(
		"Trigger Schedule",
		fmt.Sprintf("Trigger schedule %s now?", s.ID),
		command,
	).SetOnConfirm(func() {
		sl.executeTriggerSchedule(s.ID)
	}).SetOnCancel(func() {
		sl.closeModal("confirm-trigger")
	})

	sl.app.UI().Pages().AddPage("confirm-trigger", modal, true, true)
	sl.app.UI().SetFocus(modal)
}

func (sl *ScheduleList) executeTriggerSchedule(scheduleID string) {
	provider := sl.app.Provider()
	if provider == nil {
		sl.closeModal("confirm-trigger")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.TriggerSchedule(ctx, sl.namespace, scheduleID)

		sl.app.UI().QueueUpdateDraw(func() {
			sl.closeModal("confirm-trigger")
			if err != nil {
				sl.showError(err)
			} else {
				sl.loadData()
			}
		})
	}()
}

func (sl *ScheduleList) showDeleteConfirm() {
	s := sl.getSelectedSchedule()
	if s == nil {
		return
	}

	command := fmt.Sprintf(`temporal schedule delete \
  --schedule-id %s \
  --namespace %s`,
		s.ID, sl.namespace)

	modal := ui.NewConfirmModal(
		"Delete Schedule",
		fmt.Sprintf("Delete schedule %s?", s.ID),
		command,
	).SetWarning("This will permanently delete the schedule. This cannot be undone.").
		SetOnConfirm(func() {
			sl.executeDeleteSchedule(s.ID)
		}).SetOnCancel(func() {
		sl.closeModal("confirm-delete-schedule")
	})

	sl.app.UI().Pages().AddPage("confirm-delete-schedule", modal, true, true)
	sl.app.UI().SetFocus(modal)
}

func (sl *ScheduleList) executeDeleteSchedule(scheduleID string) {
	provider := sl.app.Provider()
	if provider == nil {
		sl.closeModal("confirm-delete-schedule")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.DeleteSchedule(ctx, sl.namespace, scheduleID)

		sl.app.UI().QueueUpdateDraw(func() {
			sl.closeModal("confirm-delete-schedule")
			if err != nil {
				sl.showError(err)
			} else {
				sl.loadData()
			}
		})
	}()
}

func (sl *ScheduleList) closeModal(name string) {
	sl.app.UI().Pages().RemovePage(name)
	if current := sl.app.UI().Pages().Current(); current != nil {
		sl.app.UI().SetFocus(current)
	}
}

// Name returns the view name.
func (sl *ScheduleList) Name() string {
	return "schedules"
}

// Start is called when the view becomes active.
func (sl *ScheduleList) Start() {
	sl.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'r':
			sl.loadData()
			return nil
		case 'p':
			sl.togglePreview()
			return nil
		case 'P': // Pause/Unpause toggle
			sl.showPauseConfirm()
			return nil
		case 't': // Trigger
			sl.showTriggerConfirm()
			return nil
		case 'D': // Delete
			sl.showDeleteConfirm()
			return nil
		}
		return event
	})
	sl.loadData()
}

// Stop is called when the view is deactivated.
func (sl *ScheduleList) Stop() {
	sl.table.SetInputCapture(nil)
	if sl.unsubscribeTheme != nil {
		sl.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	sl.table.Destroy()
	sl.leftPanel.Destroy()
	sl.rightPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (sl *ScheduleList) Hints() []ui.KeyHint {
	hints := []ui.KeyHint{
		{Key: "r", Description: "Refresh"},
		{Key: "j/k", Description: "Navigate"},
		{Key: "p", Description: "Preview"},
		{Key: "P", Description: "Pause/Unpause"},
		{Key: "t", Description: "Trigger"},
		{Key: "D", Description: "Delete"},
		{Key: "T", Description: "Theme"},
		{Key: "esc", Description: "Back"},
	}
	return hints
}

// Focus sets focus to the table.
func (sl *ScheduleList) Focus(delegate func(p tview.Primitive)) {
	delegate(sl.table)
}

// Draw applies theme colors dynamically and draws the view.
func (sl *ScheduleList) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	sl.SetBackgroundColor(bg)
	sl.preview.SetBackgroundColor(bg)
	sl.preview.SetTextColor(ui.ColorFg())
	sl.Flex.Draw(screen)
}
