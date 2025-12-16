package view

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atterpac/loom/internal/config"
	"github.com/atterpac/loom/internal/temporal"
	"github.com/atterpac/loom/internal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NamespaceList displays a list of Temporal namespaces with a preview panel.
type NamespaceList struct {
	*tview.Flex
	table            *ui.Table
	leftPanel        *ui.Panel
	rightPanel       *ui.Panel
	preview          *tview.TextView
	emptyState       *ui.EmptyState
	app              *App
	namespaces       []temporal.Namespace
	loading          bool
	autoRefresh      bool
	showPreview      bool
	refreshTicker    *time.Ticker
	stopRefresh      chan struct{}
	unsubscribeTheme func()
}

// NewNamespaceList creates a new namespace list view.
func NewNamespaceList(app *App) *NamespaceList {
	nl := &NamespaceList{
		Flex:        tview.NewFlex().SetDirection(tview.FlexColumn),
		table:       ui.NewTable(),
		preview:     tview.NewTextView(),
		app:         app,
		namespaces:  []temporal.Namespace{},
		showPreview: true,
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

	// Configure preview
	nl.preview.SetDynamicColors(true)
	nl.preview.SetBackgroundColor(ui.ColorBg())
	nl.preview.SetTextColor(ui.ColorFg())
	nl.preview.SetWordWrap(true)

	// Create empty state
	nl.emptyState = ui.EmptyStateNoNamespaces()

	// Create panels
	nl.leftPanel = ui.NewPanel("Namespaces")
	nl.leftPanel.SetContent(nl.table)

	nl.rightPanel = ui.NewPanel("Details")
	nl.rightPanel.SetContent(nl.preview)

	// Selection change handler to update preview
	nl.table.SetSelectionChangedFunc(func(row, col int) {
		// Adjust for header row (row 0 is header, data starts at row 1)
		dataRow := row - 1
		if dataRow >= 0 && dataRow < len(nl.namespaces) {
			nl.updatePreview(nl.namespaces[dataRow])
		}
	})

	// Selection handler - Enter navigates to workflows
	nl.table.SetOnSelect(func(row int) {
		if row >= 0 && row < len(nl.namespaces) {
			nl.app.NavigateToWorkflows(nl.namespaces[row].Name)
		}
	})

	// Register for theme changes
	nl.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		nl.SetBackgroundColor(ui.ColorBg())
		nl.preview.SetBackgroundColor(ui.ColorBg())
		nl.preview.SetTextColor(ui.ColorFg())
		// Re-render table with new colors
		if len(nl.namespaces) > 0 {
			nl.populateTable()
			// Explicitly update preview with new theme colors
			row := nl.table.SelectedRow()
			if row >= 0 && row < len(nl.namespaces) {
				nl.updatePreview(nl.namespaces[row])
			}
		}
	})

	nl.buildLayout()
}

func (nl *NamespaceList) buildLayout() {
	nl.Clear()
	if nl.showPreview {
		nl.AddItem(nl.leftPanel, 0, 3, true)
		nl.AddItem(nl.rightPanel, 0, 2, false)
	} else {
		nl.AddItem(nl.leftPanel, 0, 1, true)
	}
}

func (nl *NamespaceList) togglePreview() {
	nl.showPreview = !nl.showPreview
	nl.buildLayout()
}

func (nl *NamespaceList) updatePreview(ns temporal.Namespace) {
	stateIcon := ui.IconConnected
	stateColor := ui.TagRunning()
	if ns.State == "Deprecated" {
		stateIcon = ui.IconDisconnected
		stateColor = ui.TagFailed()
	}

	text := fmt.Sprintf(`[%s::b]Name[-:-:-]
  [%s]%s[-]

[%s::b]State[-:-:-]
  [%s]%s %s[-]

[%s::b]Retention[-:-:-]
  [%s]%s[-]

[%s::b]Description[-:-:-]
  [%s]%s[-]

[%s::b]Owner[-:-:-]
  [%s]%s[-]`,
		ui.TagFgDim(),
		ui.TagFg(), ns.Name,
		ui.TagFgDim(),
		stateColor, stateIcon, ns.State,
		ui.TagFgDim(),
		ui.TagFg(), ns.RetentionPeriod,
		ui.TagFgDim(),
		ui.TagFg(), valueOrEmpty(ns.Description, "No description"),
		ui.TagFgDim(),
		ui.TagFg(), valueOrEmpty(ns.OwnerEmail, "No owner"),
	)
	nl.preview.SetText(text)
}

func valueOrEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
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
	// Preserve current selection
	currentRow := nl.table.SelectedRow()

	nl.table.ClearRows()
	nl.table.SetHeaders("NAME", "STATE", "RETENTION")

	// Show empty state if no namespaces
	if len(nl.namespaces) == 0 {
		nl.leftPanel.SetContent(nl.emptyState)
		nl.preview.SetText("")
		return
	}

	// Show table with data
	nl.leftPanel.SetContent(nl.table)

	for _, ns := range nl.namespaces {
		nl.table.AddStyledRow(ns.State,
			ui.IconNamespace+" "+ns.Name,
			ns.State,
			ns.RetentionPeriod,
		)
	}

	if nl.table.RowCount() > 0 {
		// Restore previous selection if valid, otherwise select first row
		if currentRow >= 0 && currentRow < len(nl.namespaces) {
			nl.table.SelectRow(currentRow)
			nl.updatePreview(nl.namespaces[currentRow])
		} else {
			nl.table.SelectRow(0)
			if len(nl.namespaces) > 0 {
				nl.updatePreview(nl.namespaces[0])
			}
		}
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
		case 'p':
			nl.togglePreview()
			return nil
		case 'i':
			// Navigate to full detail view
			ns := nl.getSelectedNamespace()
			if ns != nil {
				nl.app.NavigateToNamespaceDetail(ns.Name)
			}
			return nil
		case 'n':
			nl.showCreateForm()
			return nil
		case 'e':
			nl.showEditForm()
			return nil
		case 'D':
			nl.showDeprecateConfirm()
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
	if nl.unsubscribeTheme != nil {
		nl.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	nl.table.Destroy()
	nl.leftPanel.Destroy()
	nl.rightPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (nl *NamespaceList) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "enter", Description: "Workflows"},
		{Key: "i", Description: "Info"},
		{Key: "n", Description: "Create"},
		{Key: "e", Description: "Edit"},
		{Key: "D", Description: "Deprecate"},
		{Key: "p", Description: "Preview"},
		{Key: "r", Description: "Refresh"},
		{Key: "a", Description: "Auto-refresh"},
		{Key: "T", Description: "Theme"},
		{Key: "?", Description: "Help"},
		{Key: "q", Description: "Quit"},
	}
}

// Focus sets focus to the table (which has the input handlers).
func (nl *NamespaceList) Focus(delegate func(p tview.Primitive)) {
	// If showing empty state, focus the flex container instead
	if len(nl.namespaces) == 0 {
		delegate(nl.Flex)
		return
	}
	delegate(nl.table)
}

// Draw applies theme colors dynamically and draws the view.
func (nl *NamespaceList) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	nl.SetBackgroundColor(bg)
	nl.preview.SetBackgroundColor(bg)
	nl.preview.SetTextColor(ui.ColorFg())
	nl.Flex.Draw(screen)
}

// getSelectedNamespace returns the currently selected namespace.
func (nl *NamespaceList) getSelectedNamespace() *temporal.Namespace {
	row := nl.table.SelectedRow() // Use SelectedRow() which accounts for header
	if row >= 0 && row < len(nl.namespaces) {
		return &nl.namespaces[row]
	}
	return nil
}

// CRUD Operations

func (nl *NamespaceList) showCreateForm() {
	form := ui.NewNamespaceForm()
	form.ClearFields() // Ensure it's in create mode

	form.SetOnSubmit(func(data ui.NamespaceFormData) {
		nl.closeModal("namespace-form")
		nl.showCreateConfirm(data)
	}).SetOnCancel(func() {
		nl.closeModal("namespace-form")
	})

	nl.app.UI().Pages().AddPage("namespace-form", form, true, true)
	nl.app.UI().SetFocus(form)
}

func (nl *NamespaceList) showCreateConfirm(data ui.NamespaceFormData) {
	command := fmt.Sprintf(`temporal namespace register \
  --namespace %s \
  --retention %dd`,
		data.Name, data.RetentionDays)

	if data.Description != "" {
		command += fmt.Sprintf(` \
  --description "%s"`, data.Description)
	}
	if data.OwnerEmail != "" {
		command += fmt.Sprintf(` \
  --owner-email "%s"`, data.OwnerEmail)
	}

	modal := ui.NewConfirmModal(
		"Create Namespace",
		fmt.Sprintf("Create namespace %s?", data.Name),
		command,
	).SetOnConfirm(func() {
		nl.executeCreate(data)
	}).SetOnCancel(func() {
		nl.closeModal("confirm-create")
	})

	nl.app.UI().Pages().AddPage("confirm-create", modal, true, true)
	nl.app.UI().SetFocus(modal)
}

func (nl *NamespaceList) executeCreate(data ui.NamespaceFormData) {
	provider := nl.app.Provider()
	if provider == nil {
		nl.closeModal("confirm-create")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		req := temporal.NamespaceCreateRequest{
			Name:          data.Name,
			Description:   data.Description,
			OwnerEmail:    data.OwnerEmail,
			RetentionDays: data.RetentionDays,
		}

		err := provider.CreateNamespace(ctx, req)

		nl.app.UI().QueueUpdateDraw(func() {
			nl.closeModal("confirm-create")
			if err != nil {
				nl.showError(err)
			} else {
				nl.loadData() // Refresh to show new namespace
			}
		})
	}()
}

func (nl *NamespaceList) showEditForm() {
	ns := nl.getSelectedNamespace()
	if ns == nil {
		return
	}

	// Parse retention days from string
	retentionDays := 30 // default
	if ns.RetentionPeriod != "" && ns.RetentionPeriod != "N/A" {
		parts := strings.Fields(ns.RetentionPeriod)
		if len(parts) > 0 {
			if days, err := strconv.Atoi(parts[0]); err == nil {
				retentionDays = days
			}
		}
	}

	form := ui.NewNamespaceForm()
	form.SetNamespace(ns.Name, retentionDays, ns.Description, ns.OwnerEmail)

	form.SetOnSubmit(func(data ui.NamespaceFormData) {
		nl.closeModal("namespace-form")
		nl.showUpdateConfirm(data)
	}).SetOnCancel(func() {
		nl.closeModal("namespace-form")
	})

	nl.app.UI().Pages().AddPage("namespace-form", form, true, true)
	nl.app.UI().SetFocus(form)
}

func (nl *NamespaceList) showUpdateConfirm(data ui.NamespaceFormData) {
	command := fmt.Sprintf(`temporal namespace update \
  --namespace %s \
  --retention %dd \
  --description "%s"`,
		data.Name, data.RetentionDays, data.Description)

	if data.OwnerEmail != "" {
		command += fmt.Sprintf(` \
  --owner-email "%s"`, data.OwnerEmail)
	}

	modal := ui.NewConfirmModal(
		"Update Namespace",
		fmt.Sprintf("Update namespace %s?", data.Name),
		command,
	).SetOnConfirm(func() {
		nl.executeUpdate(data)
	}).SetOnCancel(func() {
		nl.closeModal("confirm-update")
	})

	nl.app.UI().Pages().AddPage("confirm-update", modal, true, true)
	nl.app.UI().SetFocus(modal)
}

func (nl *NamespaceList) executeUpdate(data ui.NamespaceFormData) {
	provider := nl.app.Provider()
	if provider == nil {
		nl.closeModal("confirm-update")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := temporal.NamespaceUpdateRequest{
			Name:          data.Name,
			Description:   data.Description,
			OwnerEmail:    data.OwnerEmail,
			RetentionDays: data.RetentionDays,
		}

		err := provider.UpdateNamespace(ctx, req)

		nl.app.UI().QueueUpdateDraw(func() {
			nl.closeModal("confirm-update")
			if err != nil {
				nl.showError(err)
			} else {
				nl.loadData() // Refresh to show updated namespace
			}
		})
	}()
}

func (nl *NamespaceList) showDeprecateConfirm() {
	ns := nl.getSelectedNamespace()
	if ns == nil || ns.State != "Active" {
		return
	}

	command := fmt.Sprintf(`temporal namespace update \
  --namespace %s \
  --state DEPRECATED`,
		ns.Name)

	modal := ui.NewConfirmModal(
		"Deprecate Namespace",
		fmt.Sprintf("Deprecate namespace %s?", ns.Name),
		command,
	).SetWarning("Deprecated namespaces prevent new workflow executions. Existing workflows will continue. This can be reversed.").
		SetOnConfirm(func() {
			nl.executeDeprecate(ns.Name)
		}).SetOnCancel(func() {
		nl.closeModal("confirm-deprecate")
	})

	nl.app.UI().Pages().AddPage("confirm-deprecate", modal, true, true)
	nl.app.UI().SetFocus(modal)
}

func (nl *NamespaceList) executeDeprecate(name string) {
	provider := nl.app.Provider()
	if provider == nil {
		nl.closeModal("confirm-deprecate")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.DeprecateNamespace(ctx, name)

		nl.app.UI().QueueUpdateDraw(func() {
			nl.closeModal("confirm-deprecate")
			if err != nil {
				nl.showError(err)
			} else {
				nl.loadData() // Refresh to show deprecated state
			}
		})
	}()
}

func (nl *NamespaceList) closeModal(name string) {
	nl.app.UI().Pages().RemovePage(name)
	// Restore focus to current view
	if current := nl.app.UI().Pages().Current(); current != nil {
		nl.app.UI().SetFocus(current)
	}
}
