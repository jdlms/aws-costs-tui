package app

import (
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/rivo/tview"
)

func CreateApp(client *costexplorer.Client) *AppState {
	setupRosePineTheme()

	state := &AppState{
		client:    client,
		dataCache: make(map[string]CostData),
	}

	// Create components
	state.header = createHeader()
	state.footer = createFooter()
	state.mainTable = createMainTable()

	// Menu with callback to update content
	state.menu = createMenu(func(selection string) {
		updateContent(state, selection)
	})

	// Setup grid
	state.grid = setupGrid(state)

	// Create application
	state.app = tview.NewApplication().
		SetRoot(state.grid, true).
		SetFocus(state.menu)

	// Setup key bindings
	setupKeyBindings(state.app, state)

	// Show initial loading state
	initialData := CostData{
		Title: "Welcome to AWS Cost Explorer",
		Rows: [][]string{
			{"Status", "Message"},
			{"Initializing", "Loading cost data in the background..."},
			{"Tip", "Use j/k or arrow keys to navigate, Enter to select"},
		},
	}
	populateTable(state.mainTable, initialData)

	// Preload all data concurrently on startup
	go preloadAllData(state)

	return state
}
