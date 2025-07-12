package app

import (
	"fmt"
	"log"
	"sync"

	"cost-explorer/internal/aws"
	"cost-explorer/internal/types"
	"cost-explorer/internal/ui"

	"github.com/rivo/tview"
)

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
			{"Initializing", "Loading cost data..."},
		},
	}
	ui.PopulateTable(state.MainTable, initialData)

	// Load all data concurrently on startup
	go LoadAllData(state)

	return state
}

// LoadAllData fetches all cost data concurrently and stores it in memory
func LoadAllData(state *types.AppState) {
	log.Printf("Starting concurrent data loading...")

	state.App.QueueUpdateDraw(func() {
		state.Header.SetText("[yellow]Loading all cost data...[-]")
	})

	var wg sync.WaitGroup
	sections := []string{"Dashboard", "By Service", "By Usage Type", "By Region"}

	// Start all API calls concurrently
	for _, section := range sections {
		wg.Add(1)
		go func(sectionName string) {
			defer wg.Done()

			log.Printf("Fetching %s data...", sectionName)
			var data types.CostData

			switch sectionName {
			case "Dashboard":
				data = aws.GetDashboardData(state.Client)
			case "By Service":
				data = aws.GetServiceData(state.Client)
			case "By Region":
				data = aws.GetRegionData(state.Client)
			case "By Usage Type":
				data = aws.GetUsageTypeData(state.Client)
			}

			// Store data in memory
			state.CacheMutex.Lock()
			state.DataCache[sectionName] = data
			state.CacheMutex.Unlock()

			log.Printf("Loaded %s data", sectionName)
		}(section)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	log.Printf("All data loaded successfully!")

	// Show dashboard by default
	state.App.QueueUpdateDraw(func() {
		state.Header.SetText("[green]AWS Cost Explorer - Dashboard")
		state.CacheMutex.RLock()
		if dashboardData, exists := state.DataCache["Dashboard"]; exists {
			ui.PopulateTable(state.MainTable, dashboardData)
		}
		state.CacheMutex.RUnlock()
	})
}

// UpdateContent handles menu selection and updates the display
func UpdateContent(state *types.AppState, section string) {
	log.Printf("Updating content for section: %s", section)

	// Check if data is already loaded
	state.CacheMutex.RLock()
	data, exists := state.DataCache[section]
	state.CacheMutex.RUnlock()

	if exists {
		// Use already loaded data
		log.Printf("Using loaded data for %s", section)
		state.Header.SetText("[green]AWS Cost Explorer")
		ui.PopulateTable(state.MainTable, data)
	} else {
		// Data not loaded yet, show loading message and fetch asynchronously
		log.Printf("Data for %s not ready yet, fetching asynchronously", section)

		loadingData := types.CostData{
			Title: fmt.Sprintf("%s - Loading...", section),
			Rows: [][]string{
				{"Status", "Message"},
				{"Loading", "Data is being fetched..."},
			},
		}

		state.Header.SetText(fmt.Sprintf("[yellow]%s data loading...[-]", section))
		ui.PopulateTable(state.MainTable, loadingData)

		// Fetch data asynchronously to avoid blocking the UI
		go func(sectionName string) {
			log.Printf("Fetching %s data asynchronously...", sectionName)
			var data types.CostData

			switch sectionName {
			case "Dashboard":
				data = aws.GetDashboardData(state.Client)
			case "By Service":
				data = aws.GetServiceData(state.Client)
			case "By Region":
				data = aws.GetRegionData(state.Client)
			case "By Usage Type":
				data = aws.GetUsageTypeData(state.Client)
			}

			// Store data in cache
			state.CacheMutex.Lock()
			state.DataCache[sectionName] = data
			state.CacheMutex.Unlock()

			// Update UI on main thread
			state.App.QueueUpdateDraw(func() {
				// Only update if user is still on the same section
				state.CacheMutex.RLock()
				currentData, exists := state.DataCache[sectionName]
				state.CacheMutex.RUnlock()

				if exists {
					state.Header.SetText("[green]AWS Cost Explorer")
					ui.PopulateTable(state.MainTable, currentData)
					log.Printf("Updated UI with %s data", sectionName)
				}
			})
		}(section)
	}
}
