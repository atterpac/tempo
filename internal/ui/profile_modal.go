package ui

import (
	"github.com/atterpac/loom/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ProfileModal displays a modal for selecting and managing connection profiles.
type ProfileModal struct {
	*Modal
	table         *tview.Table
	nav           *TableListNavigator
	profiles      []string
	activeProfile string
	onSelect      func(name string)
	onNew         func()
	onEdit        func(name string)
	onDelete      func(name string)
}

// NewProfileModal creates a new profile selector modal.
func NewProfileModal() *ProfileModal {
	pm := &ProfileModal{
		Modal: NewModal(ModalConfig{
			Title:     "Connection Profiles",
			Width:     60,
			Height:    8,
			MinHeight: 8,
			MaxHeight: 20,
			Backdrop:  true,
		}),
		table: tview.NewTable(),
	}
	pm.setup()
	return pm
}

// SetProfiles sets the list of available profiles and the active one.
func (pm *ProfileModal) SetProfiles(profiles []string, active string) *ProfileModal {
	pm.profiles = profiles
	pm.activeProfile = active
	pm.rebuildTable()

	// Adjust modal height based on profile count
	height := len(profiles) + 2
	if height < 8 {
		height = 8
	}
	if height > 18 {
		height = 18
	}
	pm.SetSize(50, height)

	return pm
}

// SetOnSelect sets the callback when a profile is selected.
func (pm *ProfileModal) SetOnSelect(fn func(name string)) *ProfileModal {
	pm.onSelect = fn
	return pm
}

// SetOnNew sets the callback when creating a new profile.
func (pm *ProfileModal) SetOnNew(fn func()) *ProfileModal {
	pm.onNew = fn
	return pm
}

// SetOnEdit sets the callback when editing a profile.
func (pm *ProfileModal) SetOnEdit(fn func(name string)) *ProfileModal {
	pm.onEdit = fn
	return pm
}

// SetOnDelete sets the callback when deleting a profile.
func (pm *ProfileModal) SetOnDelete(fn func(name string)) *ProfileModal {
	pm.onDelete = fn
	return pm
}

// SetOnClose sets the callback when the modal is closed.
func (pm *ProfileModal) SetOnClose(fn func()) *ProfileModal {
	pm.Modal.SetOnClose(fn)
	return pm
}

func (pm *ProfileModal) setup() {
	pm.table.SetBackgroundColor(ColorBg())
	pm.table.SetSelectable(true, false)
	pm.table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(ColorBg()).
		Background(ColorAccent()))

	// No header rows in profile list
	pm.nav = NewTableListNavigator(pm.table, 0)

	pm.SetContent(pm.table)
	pm.SetHints([]KeyHint{
		{Key: "j/k", Description: "Nav"},
		{Key: "Enter", Description: "Select"},
		{Key: "n", Description: "New"},
		{Key: "e", Description: "Edit"},
		{Key: "d", Description: "Del"},
		{Key: "Esc", Description: "Close"},
	})

	// Register for theme changes
	OnThemeChange(func(_ *config.ParsedTheme) {
		pm.table.SetBackgroundColor(ColorBg())
		pm.table.SetSelectedStyle(tcell.StyleDefault.
			Foreground(ColorBg()).
			Background(ColorAccent()))
		pm.rebuildTable()
	})

	pm.rebuildTable()
}

func (pm *ProfileModal) rebuildTable() {
	pm.table.Clear()

	// Add profiles to table
	for i, name := range pm.profiles {
		marker := "  "
		if name == pm.activeProfile {
			marker = IconCompleted + " "
		}
		cell := tview.NewTableCell(marker + name).
			SetTextColor(ColorFg()).
			SetBackgroundColor(ColorBg())
		pm.table.SetCell(i, 0, cell)
	}

	// Select active profile row
	for i, name := range pm.profiles {
		if name == pm.activeProfile {
			pm.table.Select(i, 0)
			break
		}
	}
}

// GetSelectedProfile returns the currently highlighted profile name.
func (pm *ProfileModal) GetSelectedProfile() string {
	idx := pm.nav.GetSelectedIndex()
	if idx >= 0 && idx < len(pm.profiles) {
		return pm.profiles[idx]
	}
	return ""
}

// InputHandler handles keyboard input.
func (pm *ProfileModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return pm.Flex.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyEnter:
			if pm.onSelect != nil {
				selected := pm.GetSelectedProfile()
				if selected != "" {
					pm.onSelect(selected)
				}
			}
		case tcell.KeyEscape:
			pm.Close()
		case tcell.KeyUp:
			pm.nav.MoveUp()
		case tcell.KeyDown:
			pm.nav.MoveDown()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				pm.nav.MoveDown()
			case 'k':
				pm.nav.MoveUp()
			case 'n':
				if pm.onNew != nil {
					pm.onNew()
				}
			case 'e':
				if pm.onEdit != nil {
					selected := pm.GetSelectedProfile()
					if selected != "" {
						pm.onEdit(selected)
					}
				}
			case 'd':
				if pm.onDelete != nil {
					selected := pm.GetSelectedProfile()
					if selected != "" && selected != pm.activeProfile {
						pm.onDelete(selected)
					}
				}
			case 'q':
				pm.Close()
			}
		}
	})
}

// Focus delegates focus to the table.
func (pm *ProfileModal) Focus(delegate func(p tview.Primitive)) {
	delegate(pm.table)
}
