package components

import "github.com/rivo/tview"

func getMenuItems() []string {
	return []string{
		"Dashboard",
		"Current Month",
		"Forecast",
		"By Service",
		"By Region",
		"By Usage Type",
	}
}

func createMenu(onSelect func(string)) *tview.List {
	menu := tview.NewList()
	menu.SetBorder(true).SetTitle("💸 Cost Explorer")

	for _, item := range getMenuItems() {
		menu.AddItem(item, "", 0, nil)
	}

	// Note: We handle selection manually in setupKeyBindings to avoid conflicts
	// with custom j/k navigation

	return menu
}
