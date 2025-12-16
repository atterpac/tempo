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

// NamespaceDetail displays detailed information about a namespace.
type NamespaceDetail struct {
	*tview.Flex
	app              *App
	namespace        string
	detail           *temporal.NamespaceDetail
	loading          bool
	unsubscribeTheme func()

	// UI components
	infoPanel     *ui.Panel
	archivalPanel *ui.Panel
	clusterPanel  *ui.Panel
	infoView      *tview.TextView
	archivalView  *tview.TextView
	clusterView   *tview.TextView
}

// NewNamespaceDetail creates a new namespace detail view.
func NewNamespaceDetail(app *App, namespace string) *NamespaceDetail {
	nd := &NamespaceDetail{
		Flex:      tview.NewFlex().SetDirection(tview.FlexColumn),
		app:       app,
		namespace: namespace,
	}
	nd.setup()
	return nd
}

func (nd *NamespaceDetail) setup() {
	nd.SetBackgroundColor(ui.ColorBg())

	// Info view
	nd.infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	nd.infoView.SetBackgroundColor(ui.ColorBg())

	// Archival view
	nd.archivalView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	nd.archivalView.SetBackgroundColor(ui.ColorBg())

	// Cluster view
	nd.clusterView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	nd.clusterView.SetBackgroundColor(ui.ColorBg())

	// Create panels
	nd.infoPanel = ui.NewPanel("Namespace Info")
	nd.infoPanel.SetContent(nd.infoView)

	nd.archivalPanel = ui.NewPanel("Archival Configuration")
	nd.archivalPanel.SetContent(nd.archivalView)

	nd.clusterPanel = ui.NewPanel("Cluster & Replication")
	nd.clusterPanel.SetContent(nd.clusterView)

	// Left side: Info panel
	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	leftFlex.SetBackgroundColor(ui.ColorBg())
	leftFlex.AddItem(nd.infoPanel, 0, 2, false)

	// Right side: Archival + Cluster stacked
	rightFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	rightFlex.SetBackgroundColor(ui.ColorBg())
	rightFlex.AddItem(nd.archivalPanel, 0, 1, false)
	rightFlex.AddItem(nd.clusterPanel, 0, 1, false)

	// Main layout
	nd.AddItem(leftFlex, 0, 1, true)
	nd.AddItem(rightFlex, 0, 1, false)

	// Show loading state initially
	nd.infoView.SetText(fmt.Sprintf("\n [%s]Loading...[-]", ui.TagFgDim()))

	// Register for theme changes
	nd.unsubscribeTheme = ui.OnThemeChange(func(_ *config.ParsedTheme) {
		nd.SetBackgroundColor(ui.ColorBg())
		leftFlex.SetBackgroundColor(ui.ColorBg())
		rightFlex.SetBackgroundColor(ui.ColorBg())
		nd.infoView.SetBackgroundColor(ui.ColorBg())
		nd.archivalView.SetBackgroundColor(ui.ColorBg())
		nd.clusterView.SetBackgroundColor(ui.ColorBg())
		// Re-render with new colors
		if nd.detail != nil {
			nd.render()
		}
	})
}

func (nd *NamespaceDetail) loadData() {
	provider := nd.app.Provider()
	if provider == nil {
		nd.loadMockData()
		return
	}

	nd.loading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		detail, err := provider.DescribeNamespace(ctx, nd.namespace)

		nd.app.UI().QueueUpdateDraw(func() {
			nd.loading = false
			if err != nil {
				nd.showError(err)
				return
			}
			nd.detail = detail
			nd.render()
		})
	}()
}

func (nd *NamespaceDetail) loadMockData() {
	nd.detail = &temporal.NamespaceDetail{
		Namespace: temporal.Namespace{
			Name:            nd.namespace,
			State:           "Active",
			RetentionPeriod: "30 days",
			Description:     "Mock namespace for development",
			OwnerEmail:      "dev@example.com",
		},
		ID:                 "mock-namespace-id-12345",
		IsGlobalNamespace:  false,
		FailoverVersion:    0,
		HistoryArchival:    "Disabled",
		VisibilityArchival: "Disabled",
		Clusters:           []string{"active"},
	}
	nd.render()
}

func (nd *NamespaceDetail) showError(err error) {
	nd.infoView.SetText(fmt.Sprintf("\n [%s]Error: %s[-]", ui.TagFailed(), err.Error()))
	nd.archivalView.SetText("")
	nd.clusterView.SetText("")
}

func (nd *NamespaceDetail) render() {
	if nd.detail == nil {
		nd.infoView.SetText(fmt.Sprintf(" [%s]Namespace not found[-]", ui.TagFailed()))
		return
	}

	d := nd.detail
	stateColor := nd.stateColorTag(d.State)
	stateIcon := nd.stateIcon(d.State)

	// Main namespace info
	infoText := fmt.Sprintf(`
[%s::b]Name[-:-:-]           [%s]%s[-]
[%s::b]State[-:-:-]          [%s]%s %s[-]
[%s::b]Retention[-:-:-]      [%s]%s[-]
[%s::b]Description[-:-:-]    [%s]%s[-]
[%s::b]Owner Email[-:-:-]    [%s]%s[-]
[%s::b]Namespace ID[-:-:-]   [%s]%s[-]`,
		ui.TagFgDim(), ui.TagFg(), d.Name,
		ui.TagFgDim(), stateColor, stateIcon, d.State,
		ui.TagFgDim(), ui.TagFg(), d.RetentionPeriod,
		ui.TagFgDim(), ui.TagFg(), nd.valueOrNA(d.Description),
		ui.TagFgDim(), ui.TagFg(), nd.valueOrNA(d.OwnerEmail),
		ui.TagFgDim(), ui.TagFgDim(), nd.valueOrNA(d.ID),
	)
	nd.infoView.SetText(infoText)

	// Archival configuration
	archivalText := fmt.Sprintf(`
[%s::b]History Archival[-:-:-]
  [%s]%s[-]

[%s::b]Visibility Archival[-:-:-]
  [%s]%s[-]`,
		ui.TagFgDim(), ui.TagFg(), nd.valueOrNA(d.HistoryArchival),
		ui.TagFgDim(), ui.TagFg(), nd.valueOrNA(d.VisibilityArchival),
	)
	nd.archivalView.SetText(archivalText)

	// Cluster info
	globalStr := "No"
	if d.IsGlobalNamespace {
		globalStr = "Yes"
	}

	clustersStr := "None"
	if len(d.Clusters) > 0 {
		clustersStr = strings.Join(d.Clusters, ", ")
	}

	clusterText := fmt.Sprintf(`
[%s::b]Global Namespace[-:-:-]  [%s]%s[-]
[%s::b]Failover Version[-:-:-]  [%s]%d[-]
[%s::b]Clusters[-:-:-]          [%s]%s[-]`,
		ui.TagFgDim(), ui.TagFg(), globalStr,
		ui.TagFgDim(), ui.TagFg(), d.FailoverVersion,
		ui.TagFgDim(), ui.TagFg(), clustersStr,
	)
	nd.clusterView.SetText(clusterText)
}

func (nd *NamespaceDetail) valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func (nd *NamespaceDetail) stateColorTag(state string) string {
	switch state {
	case "Active":
		return ui.TagRunning()
	case "Deprecated":
		return ui.TagFailed()
	case "Deleted":
		return ui.TagFgDim()
	default:
		return ui.TagFg()
	}
}

func (nd *NamespaceDetail) stateIcon(state string) string {
	switch state {
	case "Active":
		return "●"
	case "Deprecated":
		return "○"
	case "Deleted":
		return "×"
	default:
		return "?"
	}
}

// Name returns the view name.
func (nd *NamespaceDetail) Name() string {
	return "namespace-detail"
}

// Start is called when the view becomes active.
func (nd *NamespaceDetail) Start() {
	nd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'r':
			nd.loadData()
			return nil
		case 'e':
			nd.showEditForm()
			return nil
		case 'D':
			nd.showDeprecateConfirm()
			return nil
		}
		return event
	})
	nd.loadData()
}

// Stop is called when the view is deactivated.
func (nd *NamespaceDetail) Stop() {
	nd.SetInputCapture(nil)
	if nd.unsubscribeTheme != nil {
		nd.unsubscribeTheme()
	}
	// Clean up component theme listeners to prevent memory leaks and visual glitches
	nd.infoPanel.Destroy()
	nd.archivalPanel.Destroy()
	nd.clusterPanel.Destroy()
}

// Hints returns keybinding hints for this view.
func (nd *NamespaceDetail) Hints() []ui.KeyHint {
	hints := []ui.KeyHint{
		{Key: "r", Description: "Refresh"},
		{Key: "e", Description: "Edit"},
	}

	// Only show deprecate for active namespaces
	if nd.detail != nil && nd.detail.State == "Active" {
		hints = append(hints, ui.KeyHint{Key: "D", Description: "Deprecate"})
	}

	hints = append(hints,
		ui.KeyHint{Key: "T", Description: "Theme"},
		ui.KeyHint{Key: "esc", Description: "Back"},
	)

	return hints
}

// Focus sets focus to this view.
func (nd *NamespaceDetail) Focus(delegate func(p tview.Primitive)) {
	delegate(nd.Flex)
}

// Draw applies theme colors dynamically and draws the view.
func (nd *NamespaceDetail) Draw(screen tcell.Screen) {
	bg := ui.ColorBg()
	nd.SetBackgroundColor(bg)
	nd.infoView.SetBackgroundColor(bg)
	nd.archivalView.SetBackgroundColor(bg)
	nd.clusterView.SetBackgroundColor(bg)
	nd.Flex.Draw(screen)
}

// Edit functionality

func (nd *NamespaceDetail) showEditForm() {
	if nd.detail == nil {
		return
	}

	// Parse retention days from string
	retentionDays := 30 // default
	if nd.detail.RetentionPeriod != "" && nd.detail.RetentionPeriod != "N/A" {
		parts := strings.Fields(nd.detail.RetentionPeriod)
		if len(parts) > 0 {
			if days, err := strconv.Atoi(parts[0]); err == nil {
				retentionDays = days
			}
		}
	}

	form := ui.NewNamespaceForm()
	form.SetNamespace(nd.detail.Name, retentionDays, nd.detail.Description, nd.detail.OwnerEmail)

	form.SetOnSubmit(func(data ui.NamespaceFormData) {
		nd.closeModal("namespace-form")
		nd.showUpdateConfirm(data)
	}).SetOnCancel(func() {
		nd.closeModal("namespace-form")
	})

	nd.app.UI().Pages().AddPage("namespace-form", form, true, true)
	nd.app.UI().SetFocus(form)
}

func (nd *NamespaceDetail) showUpdateConfirm(data ui.NamespaceFormData) {
	command := fmt.Sprintf(`temporal namespace update \
  --namespace %s \
  --retention %dd \
  --description "%s" \
  --owner-email "%s"`,
		data.Name, data.RetentionDays, data.Description, data.OwnerEmail)

	modal := ui.NewConfirmModal(
		"Update Namespace",
		fmt.Sprintf("Update namespace %s?", data.Name),
		command,
	).SetOnConfirm(func() {
		nd.executeUpdate(data)
	}).SetOnCancel(func() {
		nd.closeModal("confirm-update")
	})

	nd.app.UI().Pages().AddPage("confirm-update", modal, true, true)
	nd.app.UI().SetFocus(modal)
}

func (nd *NamespaceDetail) executeUpdate(data ui.NamespaceFormData) {
	provider := nd.app.Provider()
	if provider == nil {
		nd.closeModal("confirm-update")
		nd.showError(fmt.Errorf("no provider connected"))
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

		nd.app.UI().QueueUpdateDraw(func() {
			nd.closeModal("confirm-update")
			if err != nil {
				nd.showError(err)
			} else {
				nd.loadData() // Refresh to show updated values
			}
		})
	}()
}

// Deprecate functionality

func (nd *NamespaceDetail) showDeprecateConfirm() {
	if nd.detail == nil || nd.detail.State != "Active" {
		return
	}

	command := fmt.Sprintf(`temporal namespace update \
  --namespace %s \
  --state DEPRECATED`,
		nd.namespace)

	modal := ui.NewConfirmModal(
		"Deprecate Namespace",
		fmt.Sprintf("Deprecate namespace %s?", nd.namespace),
		command,
	).SetWarning("Deprecated namespaces prevent new workflow executions. Existing workflows will continue. This can be reversed by updating the namespace state.").
		SetOnConfirm(func() {
			nd.executeDeprecate()
		}).SetOnCancel(func() {
		nd.closeModal("confirm-deprecate")
	})

	nd.app.UI().Pages().AddPage("confirm-deprecate", modal, true, true)
	nd.app.UI().SetFocus(modal)
}

func (nd *NamespaceDetail) executeDeprecate() {
	provider := nd.app.Provider()
	if provider == nil {
		nd.closeModal("confirm-deprecate")
		nd.showError(fmt.Errorf("no provider connected"))
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.DeprecateNamespace(ctx, nd.namespace)

		nd.app.UI().QueueUpdateDraw(func() {
			nd.closeModal("confirm-deprecate")
			if err != nil {
				nd.showError(err)
			} else {
				nd.loadData() // Refresh to show deprecated state
				// Update hints since state changed
				nd.app.UI().Menu().SetHints(nd.Hints())
			}
		})
	}()
}

func (nd *NamespaceDetail) closeModal(name string) {
	nd.app.UI().Pages().RemovePage(name)
	// Restore focus to current view
	if current := nd.app.UI().Pages().Current(); current != nil {
		nd.app.UI().SetFocus(current)
	}
}
