package view

import (
	"context"
	"time"

	"github.com/atterpac/temportui/internal/config"
	"github.com/atterpac/temportui/internal/temporal"
	"github.com/atterpac/temportui/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NamespaceList displays a list of Temporal namespaces.
type NamespaceList struct {
	*tview.Flex
	table         *ui.Table
	panel         *ui.Panel
	app           *App
	namespaces    []temporal.Namespace
	loading       bool
	autoRefresh   bool
	refreshTicker *time.Ticker
	stopRefresh   chan struct{}
}

// NewNamespaceList creates a new namespace list view.
func NewNamespaceList(app *App) *NamespaceList {
	nl := &NamespaceList{
		Flex:        tview.NewFlex().SetDirection(tview.FlexRow),
		table:       ui.NewTable(),
		app:         app,
		namespaces:  []temporal.Namespace{},
		stopRefresh: make(chan struct{}),
	}
	nl.setup()
	return nl
}

func (nl *NamespaceList) setup() {
	nl.table.SetHeaders("NAME", "STATE", "RETENTION")
	nl.table.SetBorder(false)
	nl.table.SetBackgroundColor(ui.ColorBg())
	nl.SetBackgroundColor(ui.ColorBg())

	// Create panel
	nl.panel = ui.NewPanel("Namespaces")
	nl.panel.SetContent(nl.table)

	nl.AddItem(nl.panel, 0, 1, true)

	// Selection handler
	nl.table.SetOnSelect(func(row int) {
		if row >= 0 && row < len(nl.namespaces) {
			nl.app.NavigateToWorkflows(nl.namespaces[row].Name)
		}
	})

	// Register for theme changes
	ui.OnThemeChange(func(_ *config.ParsedTheme) {
		nl.SetBackgroundColor(ui.ColorBg())
		// Re-render table with new colors
		if len(nl.namespaces) > 0 {
			nl.populateTable()
		}
	})
}

func (nl *NamespaceList) setLoading(loading bool) {
	nl.loading = loading
	// Loading state shown via breadcrumb or status, not title
}

func (nl *NamespaceList) loadData() {
	provider := nl.app.Provider()
	if provider == nil {
		// Fallback to mock data if no provider
		nl.loadMockData()
		return
	}

	nl.setLoading(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		namespaces, err := provider.ListNamespaces(ctx)

		nl.app.UI().QueueUpdateDraw(func() {
			nl.setLoading(false)
			if err != nil {
				nl.showError(err)
				return
			}
			nl.namespaces = namespaces
			nl.populateTable()
		})
	}()
}

func (nl *NamespaceList) loadMockData() {
	// Mock data fallback when no provider is configured
	nl.namespaces = []temporal.Namespace{
		{Name: "default", State: "Active", RetentionPeriod: "7 days"},
		{Name: "production", State: "Active", RetentionPeriod: "30 days"},
		{Name: "staging", State: "Active", RetentionPeriod: "3 days"},
		{Name: "development", State: "Active", RetentionPeriod: "1 day"},
		{Name: "archived", State: "Deprecated", RetentionPeriod: "90 days"},
	}
	nl.populateTable()
}

func (nl *NamespaceList) populateTable() {
	nl.table.ClearRows()
	nl.table.SetHeaders("NAME", "STATE", "RETENTION")

	for _, ns := range nl.namespaces {
		stateIcon := ui.IconConnected
		color := ui.ColorCompleted()
		if ns.State == "Deprecated" {
			stateIcon = ui.IconDisconnected
			color = ui.ColorFgDim()
		}
		nl.table.AddColoredRow(color,
			ui.IconNamespace+" "+ns.Name,
			stateIcon+" "+ns.State,
			ns.RetentionPeriod,
		)
	}

	if nl.table.RowCount() > 0 {
		nl.table.SelectRow(0)
	}
}

func (nl *NamespaceList) showError(err error) {
	nl.table.ClearRows()
	nl.table.SetHeaders("NAME", "STATE", "RETENTION")
	nl.table.AddColoredRow(ui.ColorFailed(),
		ui.IconFailed+" Error loading namespaces",
		err.Error(),
		"",
	)
}

func (nl *NamespaceList) toggleAutoRefresh() {
	nl.autoRefresh = !nl.autoRefresh
	if nl.autoRefresh {
		nl.startAutoRefresh()
	} else {
		nl.stopAutoRefresh()
	}
}

func (nl *NamespaceList) startAutoRefresh() {
	nl.refreshTicker = time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-nl.refreshTicker.C:
				nl.app.UI().QueueUpdateDraw(func() {
					nl.loadData()
				})
			case <-nl.stopRefresh:
				return
			}
		}
	}()
}

func (nl *NamespaceList) stopAutoRefresh() {
	if nl.refreshTicker != nil {
		nl.refreshTicker.Stop()
		nl.refreshTicker = nil
	}
	// Signal stop to the goroutine
	select {
	case nl.stopRefresh <- struct{}{}:
	default:
	}
}

// Name returns the view name.
func (nl *NamespaceList) Name() string {
	return "namespaces"
}

// Start is called when the view becomes active.
func (nl *NamespaceList) Start() {
	nl.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			nl.app.UI().Stop()
			return nil
		case 'a':
			nl.toggleAutoRefresh()
			return nil
		case 'r':
			nl.loadData()
			return nil
		}
		return event
	})
	// Load data when view becomes active
	nl.loadData()
}

// Stop is called when the view is deactivated.
func (nl *NamespaceList) Stop() {
	nl.table.SetInputCapture(nil)
	nl.stopAutoRefresh()
}

// Hints returns keybinding hints for this view.
func (nl *NamespaceList) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "enter", Description: "Select"},
		{Key: "r", Description: "Refresh"},
		{Key: "a", Description: "Auto-refresh"},
		{Key: "j/k", Description: "Navigate"},
		{Key: "q", Description: "Quit"},
		{Key: "?", Description: "Help"},
	}
}
