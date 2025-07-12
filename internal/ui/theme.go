package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SetupRosePineTheme configures the Rose Pine color theme for the TUI
func SetupRosePineTheme() {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    tcell.NewRGBColor(35, 33, 54),    // base (#232136)
		ContrastBackgroundColor:     tcell.NewRGBColor(42, 39, 63),    // surface (#2a273f)
		MoreContrastBackgroundColor: tcell.NewRGBColor(57, 53, 82),    // overlay (#393552)
		BorderColor:                 tcell.NewRGBColor(110, 106, 134), // muted (#6e6a86)
		TitleColor:                  tcell.NewRGBColor(235, 188, 186), // rose (#ebbcba)
		GraphicsColor:               tcell.NewRGBColor(156, 207, 216), // foam (#9ccfd8)
		PrimaryTextColor:            tcell.NewRGBColor(224, 222, 244), // text (#e0def4)
		SecondaryTextColor:          tcell.NewRGBColor(144, 140, 170), // subtle (#908caa)
		TertiaryTextColor:           tcell.NewRGBColor(110, 106, 134), // muted (#6e6a86)
		InverseTextColor:            tcell.NewRGBColor(35, 33, 54),    // base (#232136)
		ContrastSecondaryTextColor:  tcell.NewRGBColor(224, 222, 244), // text (#e0def4)
	}
}
