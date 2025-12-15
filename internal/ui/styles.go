package ui

import (
	"sync"

	"github.com/atterpac/temportui/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	activeTheme    *config.ParsedTheme
	themeListeners []func(*config.ParsedTheme)
	themeMu        sync.RWMutex
)

// InitTheme initializes the theme system with the given theme name.
// Must be called before any UI components are created.
func InitTheme(name string) error {
	theme, err := config.LoadTheme(name)
	if err != nil {
		return err
	}

	themeMu.Lock()
	activeTheme = theme
	themeMu.Unlock()

	applyGlobalStyles()
	return nil
}

// SetTheme switches to a new theme and notifies all listeners.
func SetTheme(name string) error {
	theme, err := config.LoadTheme(name)
	if err != nil {
		return err
	}

	themeMu.Lock()
	activeTheme = theme
	listeners := make([]func(*config.ParsedTheme), len(themeListeners))
	copy(listeners, themeListeners)
	themeMu.Unlock()

	applyGlobalStyles()

	// Notify all registered listeners
	for _, fn := range listeners {
		fn(theme)
	}

	return nil
}

// OnThemeChange registers a callback to be called when the theme changes.
// Returns an unsubscribe function.
func OnThemeChange(fn func(*config.ParsedTheme)) func() {
	themeMu.Lock()
	themeListeners = append(themeListeners, fn)
	index := len(themeListeners) - 1
	themeMu.Unlock()

	// Return unsubscribe function
	return func() {
		themeMu.Lock()
		defer themeMu.Unlock()
		if index < len(themeListeners) {
			themeListeners = append(themeListeners[:index], themeListeners[index+1:]...)
		}
	}
}

// ActiveTheme returns the current active theme.
func ActiveTheme() *config.ParsedTheme {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return activeTheme
}

// applyGlobalStyles sets the global tview styles from the active theme.
func applyGlobalStyles() {
	themeMu.RLock()
	t := activeTheme
	themeMu.RUnlock()

	if t == nil {
		return
	}

	tview.Styles.PrimitiveBackgroundColor = t.Colors.Bg
	tview.Styles.ContrastBackgroundColor = t.Colors.BgLight
	tview.Styles.MoreContrastBackgroundColor = t.Colors.BgDark
	tview.Styles.BorderColor = t.Colors.Border
	tview.Styles.TitleColor = t.Colors.Accent
	tview.Styles.PrimaryTextColor = t.Colors.Fg
	tview.Styles.SecondaryTextColor = t.Colors.FgDim
}

// Color getters - return tcell.Color from active theme

func ColorBg() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Bg
}

func ColorBgLight() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.BgLight
}

func ColorBgDark() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.BgDark
}

func ColorFg() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Fg
}

func ColorFgDim() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.FgDim
}

func ColorBorder() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Border
}

func ColorHighlight() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Highlight
}

func ColorAccent() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Accent
}

func ColorAccentDim() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.AccentDim
}

func ColorRunning() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Running
}

func ColorCompleted() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Completed
}

func ColorFailed() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Failed
}

func ColorCanceled() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Canceled
}

func ColorTerminated() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Terminated
}

func ColorTimedOut() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.TimedOut
}

func ColorHeader() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Header
}

func ColorMenu() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Menu
}

func ColorTableHdr() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.TableHeader
}

func ColorKey() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Key
}

func ColorCrumb() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.Crumb
}

func ColorPanelBorder() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.PanelBorder
}

func ColorPanelTitle() tcell.Color {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return tcell.ColorDefault
	}
	return activeTheme.Colors.PanelTitle
}

// Tag getters - return hex strings for tview color tags

func TagBg() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#1e1e2e"
	}
	return activeTheme.Tags.Bg
}

func TagFg() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#cdd6f4"
	}
	return activeTheme.Tags.Fg
}

func TagFgDim() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#6c7086"
	}
	return activeTheme.Tags.FgDim
}

func TagAccent() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f5c2e7"
	}
	return activeTheme.Tags.Accent
}

func TagKey() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#cba6f7"
	}
	return activeTheme.Tags.Key
}

func TagCrumb() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f5c2e7"
	}
	return activeTheme.Tags.Crumb
}

func TagTableHdr() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f5c2e7"
	}
	return activeTheme.Tags.TableHeader
}

func TagHighlight() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#585b70"
	}
	return activeTheme.Tags.Highlight
}

func TagBorder() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#45475a"
	}
	return activeTheme.Tags.Border
}

func TagRunning() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f9e2af"
	}
	return activeTheme.Tags.Running
}

func TagCompleted() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#a6e3a1"
	}
	return activeTheme.Tags.Completed
}

func TagFailed() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f38ba8"
	}
	return activeTheme.Tags.Failed
}

func TagCanceled() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#fab387"
	}
	return activeTheme.Tags.Canceled
}

func TagPanelBorder() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#585b70"
	}
	return activeTheme.Tags.PanelBorder
}

func TagPanelTitle() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme == nil {
		return "#f5c2e7"
	}
	return activeTheme.Tags.PanelTitle
}

// Nerd Font icons (theme-agnostic)
const (
	// Status icons
	IconRunning    = "\uf144" // nf-fa-play_circle
	IconCompleted  = "\uf00c" // nf-fa-check
	IconFailed     = "\uf00d" // nf-fa-times
	IconCanceled   = "\uf05e" // nf-fa-ban
	IconTerminated = "\uf28d" // nf-fa-stop_circle
	IconTimedOut   = "\uf017" // nf-fa-clock_o
	IconPending    = "\uf10c" // nf-fa-circle_o

	// Navigation
	IconArrowRight = "\uf054" // nf-fa-chevron_right
	IconArrowDown  = "\uf078" // nf-fa-chevron_down
	IconArrowUp    = "\uf077" // nf-fa-chevron_up
	IconBullet     = "\uf192" // nf-fa-dot_circle_o
	IconDot        = "\uf111" // nf-fa-circle

	// Separators
	IconSeparator = "\uf105" // nf-fa-angle_right
	IconDash      = "\uf068" // nf-fa-minus

	// Indicators
	IconConnected    = "\uf1e6" // nf-fa-plug
	IconDisconnected = "\uf127" // nf-fa-chain_broken
	IconActivity     = "\uf013" // nf-fa-cog
	IconWorkflow     = "\uf0e7" // nf-fa-bolt
	IconNamespace    = "\uf0e8" // nf-fa-sitemap
	IconTaskQueue    = "\uf0ae" // nf-fa-tasks
	IconEvent        = "\uf1da" // nf-fa-history

	// Box drawing
	BoxTopLeft     = "\u256d"
	BoxTopRight    = "\u256e"
	BoxBottomLeft  = "\u2570"
	BoxBottomRight = "\u256f"
	BoxHorizontal  = "\u2500"
	BoxVertical    = "\u2502"
)

// Logo for the header
const Logo = `temporal-tui`

// LogoSmall is a compact version
const LogoSmall = "temporal-tui"

// StatusIcon returns the icon for a workflow status.
func StatusIcon(status string) string {
	switch status {
	case "Running":
		return IconRunning
	case "Completed":
		return IconCompleted
	case "Failed":
		return IconFailed
	case "Canceled":
		return IconCanceled
	case "Terminated":
		return IconTerminated
	case "TimedOut":
		return IconTimedOut
	default:
		return IconPending
	}
}

// StatusColorTcell returns the tcell color for a workflow status.
func StatusColorTcell(status string) tcell.Color {
	switch status {
	case "Running":
		return ColorRunning()
	case "Completed":
		return ColorCompleted()
	case "Failed":
		return ColorFailed()
	case "Canceled":
		return ColorCanceled()
	case "Terminated":
		return ColorTerminated()
	case "TimedOut":
		return ColorTimedOut()
	default:
		return ColorFg()
	}
}

// StatusColorTag returns the tview color tag for a status.
func StatusColorTag(status string) string {
	themeMu.RLock()
	defer themeMu.RUnlock()

	if activeTheme == nil {
		// Fallback to catppuccin mocha defaults
		switch status {
		case "Running":
			return "#f9e2af"
		case "Completed":
			return "#a6e3a1"
		case "Failed":
			return "#f38ba8"
		case "Canceled":
			return "#fab387"
		case "Terminated":
			return "#cba6f7"
		case "TimedOut":
			return "#f38ba8"
		default:
			return "#cdd6f4"
		}
	}

	switch status {
	case "Running":
		return activeTheme.Tags.Running
	case "Completed":
		return activeTheme.Tags.Completed
	case "Failed":
		return activeTheme.Tags.Failed
	case "Canceled":
		return activeTheme.Tags.Canceled
	case "Terminated":
		return activeTheme.Tags.Terminated
	case "TimedOut":
		return activeTheme.Tags.TimedOut
	default:
		return activeTheme.Tags.Fg
	}
}
