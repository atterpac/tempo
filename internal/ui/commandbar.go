package ui

import (
	"github.com/atterpac/temportui/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// CommandType identifies the type of command being entered.
type CommandType int

const (
	CommandNone CommandType = iota
	CommandFilter
	CommandAction // For future : commands
)

// CommandBar provides a k9s-style command/filter input bar with matching StatsBar styling.
type CommandBar struct {
	*tview.Box
	input       *tview.InputField
	active      bool
	commandType CommandType
	text        string
	cursorPos   int
	onSubmit    func(cmd CommandType, text string)
	onCancel    func()
	onChange    func(text string)
}

// NewCommandBar creates a new command bar component.
func NewCommandBar() *CommandBar {
	cb := &CommandBar{
		Box:   tview.NewBox(),
		input: tview.NewInputField(),
	}
	cb.SetBackgroundColor(ColorBg())

	// Register for theme changes
	OnThemeChange(func(_ *config.ParsedTheme) {
		cb.SetBackgroundColor(ColorBg())
	})

	return cb
}

// Activate shows the command bar with the given command type.
func (cb *CommandBar) Activate(cmdType CommandType) {
	cb.active = true
	cb.commandType = cmdType
	cb.text = ""
	cb.cursorPos = 0
}

// Deactivate hides the command bar.
func (cb *CommandBar) Deactivate() {
	cb.active = false
	cb.commandType = CommandNone
	cb.text = ""
	cb.cursorPos = 0
}

// IsActive returns whether the command bar is active.
func (cb *CommandBar) IsActive() bool {
	return cb.active
}

// Type returns the current command type.
func (cb *CommandBar) Type() CommandType {
	return cb.commandType
}

// GetText returns the current input text.
func (cb *CommandBar) GetText() string {
	return cb.text
}

// SetText sets the input text.
func (cb *CommandBar) SetText(text string) {
	cb.text = text
	cb.cursorPos = len(text)
	if cb.onChange != nil {
		cb.onChange(text)
	}
}

// SetOnSubmit sets the callback for when a command is submitted.
func (cb *CommandBar) SetOnSubmit(fn func(cmd CommandType, text string)) {
	cb.onSubmit = fn
}

// SetOnCancel sets the callback for when the command bar is cancelled.
func (cb *CommandBar) SetOnCancel(fn func()) {
	cb.onCancel = fn
}

// SetOnChange sets the callback for live text changes.
func (cb *CommandBar) SetOnChange(fn func(text string)) {
	cb.onChange = fn
}

// Draw renders the command bar with the same styling as StatsBar.
func (cb *CommandBar) Draw(screen tcell.Screen) {
	cb.Box.DrawForSubclass(screen, cb)

	x, y, width, height := cb.GetInnerRect()
	if width <= 0 || height < 3 {
		return
	}

	borderStyle := tcell.StyleDefault.Foreground(ColorPanelBorder()).Background(ColorBg())
	titleStyle := tcell.StyleDefault.Foreground(ColorAccent()).Background(ColorBg()).Bold(true)
	textStyle := tcell.StyleDefault.Foreground(ColorFg()).Background(ColorBg())
	promptStyle := tcell.StyleDefault.Foreground(ColorAccent()).Background(ColorBg())

	// Draw rounded border (same as StatsBar)
	screen.SetContent(x, y, '╭', nil, borderStyle)
	screen.SetContent(x+width-1, y, '╮', nil, borderStyle)
	screen.SetContent(x, y+height-1, '╰', nil, borderStyle)
	screen.SetContent(x+width-1, y+height-1, '╯', nil, borderStyle)

	for i := x + 1; i < x+width-1; i++ {
		screen.SetContent(i, y, '─', nil, borderStyle)
		screen.SetContent(i, y+height-1, '─', nil, borderStyle)
	}

	for i := y + 1; i < y+height-1; i++ {
		screen.SetContent(x, i, '│', nil, borderStyle)
		screen.SetContent(x+width-1, i, '│', nil, borderStyle)
	}

	// Draw title in top border
	var title string
	switch cb.commandType {
	case CommandFilter:
		title = " Filter "
	case CommandAction:
		title = " Command "
	default:
		title = " Input "
	}
	titleRunes := []rune(title)
	titleX := x + 2
	for i, r := range titleRunes {
		if titleX+i >= x+width-1 {
			break
		}
		screen.SetContent(titleX+i, y, r, nil, titleStyle)
	}

	// Draw prompt and input on content line
	contentY := y + 1
	contentX := x + 2

	// Draw prompt symbol
	var prompt string
	switch cb.commandType {
	case CommandFilter:
		prompt = IconArrowRight + " /"
	case CommandAction:
		prompt = IconArrowRight + " :"
	default:
		prompt = IconArrowRight + " "
	}

	for _, r := range []rune(prompt) {
		if contentX >= x+width-2 {
			break
		}
		screen.SetContent(contentX, contentY, r, nil, promptStyle)
		contentX++
	}

	// Draw input text
	inputText := cb.text
	for i, r := range []rune(inputText) {
		if contentX+i >= x+width-2 {
			break
		}
		screen.SetContent(contentX+i, contentY, r, nil, textStyle)
	}

	// Draw cursor
	cursorX := contentX + cb.cursorPos
	if cursorX < x+width-2 {
		cursorStyle := tcell.StyleDefault.Foreground(ColorBg()).Background(ColorFg())
		if cb.cursorPos < len(cb.text) {
			r := []rune(cb.text)[cb.cursorPos]
			screen.SetContent(cursorX, contentY, r, nil, cursorStyle)
		} else {
			screen.SetContent(cursorX, contentY, ' ', nil, cursorStyle)
		}
	}

	// Draw hint on right side
	hint := "[Esc] Cancel  [Enter] Apply"
	hintStyle := tcell.StyleDefault.Foreground(ColorFgDim()).Background(ColorBg())
	hintX := x + width - len(hint) - 3
	if hintX > contentX+len(cb.text)+2 {
		for i, r := range []rune(hint) {
			screen.SetContent(hintX+i, contentY, r, nil, hintStyle)
		}
	}
}

// InputHandler handles keyboard input for the command bar.
func (cb *CommandBar) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return cb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyEnter:
			if cb.onSubmit != nil {
				cb.onSubmit(cb.commandType, cb.text)
			}
			if cb.onCancel != nil {
				cb.onCancel()
			}
		case tcell.KeyEscape:
			if cb.onCancel != nil {
				cb.onCancel()
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if cb.cursorPos > 0 {
				cb.text = cb.text[:cb.cursorPos-1] + cb.text[cb.cursorPos:]
				cb.cursorPos--
				if cb.onChange != nil {
					cb.onChange(cb.text)
				}
			}
		case tcell.KeyDelete:
			if cb.cursorPos < len(cb.text) {
				cb.text = cb.text[:cb.cursorPos] + cb.text[cb.cursorPos+1:]
				if cb.onChange != nil {
					cb.onChange(cb.text)
				}
			}
		case tcell.KeyLeft:
			if cb.cursorPos > 0 {
				cb.cursorPos--
			}
		case tcell.KeyRight:
			if cb.cursorPos < len(cb.text) {
				cb.cursorPos++
			}
		case tcell.KeyHome, tcell.KeyCtrlA:
			cb.cursorPos = 0
		case tcell.KeyEnd, tcell.KeyCtrlE:
			cb.cursorPos = len(cb.text)
		case tcell.KeyRune:
			r := event.Rune()
			cb.text = cb.text[:cb.cursorPos] + string(r) + cb.text[cb.cursorPos:]
			cb.cursorPos++
			if cb.onChange != nil {
				cb.onChange(cb.text)
			}
		}
	})
}

// Focus is called when the command bar receives focus.
func (cb *CommandBar) Focus(delegate func(p tview.Primitive)) {
	cb.Box.Focus(delegate)
}

// HasFocus returns whether the command bar has focus.
func (cb *CommandBar) HasFocus() bool {
	return cb.Box.HasFocus()
}
