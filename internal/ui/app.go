package ui

import (
	"github.com/rivo/tview"
)

// App wraps tview.Application with additional functionality.
type App struct {
	*tview.Application
	main       *tview.Flex
	statsBar   *StatsBar
	crumbs     *Crumbs
	menu       *Menu
	commandBar *CommandBar
	pages      *Pages
	content    tview.Primitive
	topBar     *tview.Flex // Holds either statsBar or command bar
}

// NewApp creates a new application wrapper.
func NewApp() *App {
	app := &App{
		Application: tview.NewApplication(),
		statsBar:    NewStatsBar(),
		crumbs:      NewCrumbs(),
		menu:        NewMenu(),
		commandBar:  NewCommandBar(),
		pages:       NewPages(),
	}
	app.buildLayout()
	return app
}

func (a *App) buildLayout() {
	// Note: Global tview.Styles are set by InitTheme() in styles.go
	// which should be called before NewApp()

	// Create top bar that can hold either statsBar or command bar
	a.topBar = tview.NewFlex().SetDirection(tview.FlexRow)
	a.topBar.AddItem(a.statsBar, 3, 0, false)
	a.topBar.SetBackgroundColor(ColorBg())

	a.main = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.topBar, 3, 0, false).
		AddItem(a.crumbs, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.menu, 1, 0, false)

	a.main.SetBackgroundColor(ColorBg())
	a.SetRoot(a.main, true)
}

// StatsBar returns the stats bar component.
func (a *App) StatsBar() *StatsBar {
	return a.statsBar
}

// Crumbs returns the breadcrumb component.
func (a *App) Crumbs() *Crumbs {
	return a.crumbs
}

// Menu returns the menu component.
func (a *App) Menu() *Menu {
	return a.menu
}

// Pages returns the pages component.
func (a *App) Pages() *Pages {
	return a.pages
}

// SetContent sets the main content area (used by views).
func (a *App) SetContent(p tview.Primitive) {
	a.content = p
	a.pages.SetContent(p)
}

// CommandBar returns the command bar component.
func (a *App) CommandBar() *CommandBar {
	return a.commandBar
}

// ShowCommandBar activates the command bar, replacing the stats bar.
func (a *App) ShowCommandBar(cmdType CommandType) {
	a.topBar.Clear()
	a.topBar.AddItem(a.commandBar, 3, 0, true)
	a.commandBar.Activate(cmdType)
	a.SetFocus(a.commandBar)
}

// HideCommandBar deactivates the command bar and restores the stats bar.
func (a *App) HideCommandBar() {
	a.commandBar.Deactivate()
	a.topBar.Clear()
	a.topBar.AddItem(a.statsBar, 3, 0, false)
}

