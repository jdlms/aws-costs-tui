package ui

import (
	"github.com/rivo/tview"
)

// CreateMenu creates the main navigation menu
func CreateMenu(onSelect func(string)) *tview.List {
	menu := tview.NewList()
	menu.SetBorder(true).SetTitle("💸 Cost Explorer")

	menuItems := []string{
		"Dashboard",
		"By Service",
		"By Region",
		"By Usage Type",
	}

	for _, item := range menuItems {
		menu.AddItem(item, "", 0, nil)
	}

	// Note: We handle selection manually in key bindings to avoid conflicts
	// with custom j/k navigation
	return menu
}

// CreateMainTable creates the main data display table
func CreateMainTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true).SetTitle("Cost Data")
	table.SetSelectable(true, false) // Allow row selection but not column selection
	table.SetFixed(1, 0)             // Fix the first row as header
	return table
}

// CreateHeader creates the header text view
func CreateHeader() *tview.TextView {
	header := tview.NewTextView()
	header.SetBorder(true)
	header.SetText("AWS Cost Explorer - Loading...")
	header.SetTextAlign(tview.AlignCenter)
	header.SetDynamicColors(true)
	return header
}

// CreateFooter creates the footer text view with help text
func CreateFooter() *tview.TextView {
	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText("Press 'q' to quit | 'j/k' to navigate | Enter to select & enter table | Tab to return to menu | PgUp/PgDn to page")
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetDynamicColors(true)
	return footer
}
