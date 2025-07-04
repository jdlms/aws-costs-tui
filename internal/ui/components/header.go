package components

import "github.com/rivo/tview"

func createHeader() *tview.TextView {
	header := tview.NewTextView()
	header.SetBorder(true)
	header.SetText("AWS Cost Explorer - Loading...")
	header.SetTextAlign(tview.AlignCenter)
	header.SetDynamicColors(true)
	return header
}
