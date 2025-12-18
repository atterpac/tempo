package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Panel is a container with rounded borders and an optional title.
type Panel struct {
	*tview.Box
	content         tview.Primitive
	title           string
	titleColorOverride *tcell.Color
}

// NewPanel creates a new panel with rounded borders.
func NewPanel(title string) *Panel {
	p := &Panel{
		Box:   tview.NewBox(),
		title: title,
	}
	// Use tcell.ColorDefault to let Draw() pick up current theme colors
	p.SetBackgroundColor(tcell.ColorDefault)
	return p
}

// Destroy is a no-op kept for backward compatibility.
func (p *Panel) Destroy() {}

// SetContent sets the content primitive inside the panel.
func (p *Panel) SetContent(content tview.Primitive) *Panel {
	p.content = content
	return p
}

// SetTitle sets the panel title.
func (p *Panel) SetTitle(title string) *Panel {
	p.title = title
	return p
}

// SetBorderColor is a no-op - colors are read dynamically.
func (p *Panel) SetBorderColor(color tcell.Color) *Panel {
	return p
}

// SetTitleColor sets a custom title color, overriding the theme default.
// Pass tcell.ColorDefault to reset to theme default.
func (p *Panel) SetTitleColor(color tcell.Color) *Panel {
	if color == tcell.ColorDefault {
		p.titleColorOverride = nil
	} else {
		p.titleColorOverride = &color
	}
	return p
}

// Draw renders the panel with rounded borders.
func (p *Panel) Draw(screen tcell.Screen) {
	// Read colors dynamically at draw time
	bgColor := ColorBg()
	borderColor := ColorPanelBorder()
	titleColor := ColorPanelTitle()
	if p.titleColorOverride != nil {
		titleColor = *p.titleColorOverride
	}

	p.Box.SetBackgroundColor(bgColor)
	p.Box.DrawForSubclass(screen, p)

	x, y, width, height := p.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	borderStyle := tcell.StyleDefault.Foreground(borderColor).Background(bgColor)
	titleStyle := tcell.StyleDefault.Foreground(titleColor).Background(bgColor).Bold(true)

	// Draw corners
	screen.SetContent(x, y, '╭', nil, borderStyle)
	screen.SetContent(x+width-1, y, '╮', nil, borderStyle)
	screen.SetContent(x, y+height-1, '╰', nil, borderStyle)
	screen.SetContent(x+width-1, y+height-1, '╯', nil, borderStyle)

	// Draw horizontal borders
	for i := x + 1; i < x+width-1; i++ {
		screen.SetContent(i, y, '─', nil, borderStyle)
		screen.SetContent(i, y+height-1, '─', nil, borderStyle)
	}

	// Draw vertical borders
	for i := y + 1; i < y+height-1; i++ {
		screen.SetContent(x, i, '│', nil, borderStyle)
		screen.SetContent(x+width-1, i, '│', nil, borderStyle)
	}

	// Draw title if present
	if p.title != "" && width > 4 {
		titleText := " " + p.title + " "
		titleRunes := []rune(titleText)
		maxLen := width - 4 // Leave room for corners and padding
		if len(titleRunes) > maxLen {
			titleRunes = titleRunes[:maxLen]
		}
		titleX := x + 2
		for i, r := range titleRunes {
			screen.SetContent(titleX+i, y, r, nil, titleStyle)
		}
	}

	// Draw content inside the border
	if p.content != nil {
		// Set content bounds inside the border
		p.content.SetRect(x+1, y+1, width-2, height-2)
		p.content.Draw(screen)
	}
}

// Focus delegates focus to the content.
func (p *Panel) Focus(delegate func(p tview.Primitive)) {
	if p.content != nil {
		delegate(p.content)
	}
}

// HasFocus returns whether the content has focus.
func (p *Panel) HasFocus() bool {
	if p.content != nil {
		return p.content.HasFocus()
	}
	return false
}

// InputHandler returns the content's input handler.
func (p *Panel) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if p.content != nil {
		return p.content.InputHandler()
	}
	return nil
}

// MouseHandler returns the content's mouse handler.
func (p *Panel) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	if p.content != nil {
		return p.content.MouseHandler()
	}
	return nil
}
