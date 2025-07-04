package components

import "github.com/rivo/tview"

func createFooter() *tview.TextView {
	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText("Press 'q' to quit | 'j/k' or ↑/↓ to navigate | Enter to select")
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetDynamicColors(true)
	return footer
}
