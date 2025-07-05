package app

import (
	"fmt"
	"log"

	"github.com/rivo/tview"
	"cost-explorer/internal/cache"
	"cost-explorer/internal/types"
	"cost-explorer/internal/ui"
)

// App wraps the table for the interface
type App struct {
	state *types.AppState
}

// PopulateTable implements the TablePopulator interface
func (a *App) PopulateTable(data types.CostData) {
	ui.PopulateTable(a.state.MainTable, data)
}

// CreateApp initializes and returns the application state
func CreateApp(client *types.AppState) *types.AppState {
	ui.SetupRosePineTheme()

	state := &types.AppState{
		Client:    client.Client,
		DataCache: make(map[string]types.CostData),
	}

	// Create components
	state.Header = ui.CreateHeader()
	state.Footer = ui.CreateFooter()
	state.MainTable = ui.CreateMainTable()

	// Menu with callback to update content
	state.Menu = ui.CreateMenu(func(selection string) {
		UpdateContent(state, selection)
	})

	// Setup grid
	state.Grid = ui.SetupGrid(state)

	// Create application
	state.App = tview.NewApplication().
		SetRoot(state.Grid, true).
		SetFocus(state.Menu)

	// Setup key bindings
	SetupKeyBindings(state, UpdateContent)

	// Show initial loading state
	initialData := types.CostData{
		Title: "Welcome to AWS Cost Explorer",
		Rows: [][]string{
			{"Status", "Message"},
			{"Initializing", "Loading cost data in the background..."},
			{"Tip", "Use j/k or arrow keys to navigate, Enter to select"},
		},
	}
	ui.PopulateTable(state.MainTable, initialData)

	// Preload all data concurrently on startup
	appWrapper := &App{state: state}
	go cache.PreloadAllData(state, appWrapper)

	return state
}

// UpdateContent handles menu selection and updates the display
func UpdateContent(state *types.AppState, section string) {
	log.Printf("Starting updateContent for section: %s", section)

	// Check if data is cached
	state.CacheMutex.RLock()
	data, exists := state.DataCache[section]
	state.CacheMutex.RUnlock()

	log.Printf("Data exists for %s: %v", section, exists)

	if exists {
		// Use cached data - instant response!
		log.Printf("Using cached data for %s", section)

		// Update header first
		go func() {
			state.App.QueueUpdateDraw(func() {
				state.Header.SetText(fmt.Sprintf("[green]AWS Cost Explorer - %s[-] (Cached data)", section))
			})
		}()

		// Then update table in a separate goroutine to prevent blocking
		go func() {
			state.App.QueueUpdateDraw(func() {
				log.Printf("About to populate table for %s", section)
				ui.PopulateTable(state.MainTable, data)
				log.Printf("Finished populating table for %s", section)
			})
		}()

		log.Printf("Successfully queued UI updates for %s", section)
	} else {
		// Data not loaded yet, show loading message
		log.Printf("Data for %s not ready yet, showing loading message", section)

		emptyData := types.CostData{
			Title: fmt.Sprintf("%s - Loading...", section),
			Rows: [][]string{
				{"Status", "Message"},
				{"Loading", "Data is being fetched in the background..."},
			},
		}

		go func() {
			state.App.QueueUpdateDraw(func() {
				state.Header.SetText(fmt.Sprintf("[yellow]%s data still loading...[-]", section))
			})
		}()

		go func() {
			state.App.QueueUpdateDraw(func() {
				ui.PopulateTable(state.MainTable, emptyData)
			})
		}()
	}
}
