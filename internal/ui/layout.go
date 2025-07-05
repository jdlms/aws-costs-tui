package ui

import (
	"github.com/rivo/tview"
	"cost-explorer/internal/types"
)

// SetupGrid configures the main grid layout
func SetupGrid(state *types.AppState) *tview.Grid {
	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(25, 0).
		SetBorders(false)

	// Static items (header and footer span both columns)
	grid.AddItem(state.Header, 0, 0, 1, 2, 0, 0, false)
	grid.AddItem(state.Footer, 2, 0, 1, 2, 0, 0, false)

	// Main layout (2 columns: menu + table)
	grid.AddItem(state.Menu, 1, 0, 1, 1, 0, 80, true)
	grid.AddItem(state.MainTable, 1, 1, 1, 1, 0, 80, false)

	return grid
}
