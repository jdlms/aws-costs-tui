package components

import "github.com/rivo/tview"

func setupGrid(state *AppState) *tview.Grid {
	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(25, 0).
		SetBorders(false)

	// Static items (header and footer span both columns)
	grid.AddItem(state.header, 0, 0, 1, 2, 0, 0, false)
	grid.AddItem(state.footer, 2, 0, 1, 2, 0, 0, false)

	// Main layout (2 columns: menu + table)
	grid.AddItem(state.menu, 1, 0, 1, 1, 0, 80, true)
	grid.AddItem(state.mainTable, 1, 1, 1, 1, 0, 80, false)

	return grid
}
