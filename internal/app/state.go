// state.go - application state management
package app

import (
	"fmt"
	"log"
	"sync"
	"time"

	"cost-explorer/internal"
	"cost-explorer/internal/aws"
	"cost-explorer/internal/ui/components"
)

// Preload all data concurrently using the same pattern as main.go
func PreloadAllData(state *internal.AppState) {
	log.Printf("Starting concurrent data preload...")

	state.App.QueueUpdateDraw(func() {
		state.Header.SetText("[yellow]Loading all cost data concurrently...[-]")
	})

	// Use WaitGroup to wait for all goroutines to complete - same as main.go
	var wg sync.WaitGroup

	// Use a channel to collect results
	results := make(chan struct {
		name string
		data internal.CostData
		err  error
	}, 6)

	// Add a timeout for the entire operation
	timeout := time.After(2 * time.Minute)

	// Start all API calls concurrently - same pattern as main.go
	wg.Add(6)

	go func() {
		defer wg.Done()
		log.Printf("Fetching dashboard data...")
		data := aws.GetDashboardData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"Dashboard", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching current month data...")
		data := aws.GetCurrentMonthData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"Current Month", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching forecast data...")
		data := aws.GetForecastData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"Forecast", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching service data...")
		data := aws.GetServiceData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"By Service", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching region data...")
		data := aws.GetRegionData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"By Region", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching usage type data...")
		data := aws.GetUsageTypeData(state.Client)
		results <- struct {
			name string
			data internal.CostData
			err  error
		}{"By Usage Type", data, nil}
	}()

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and cache them with timeout protection
	loadedCount := 0
	for {
		select {
		case result, ok := <-results:
			if !ok {
				// Channel closed, all results collected
				goto completed
			}

			state.CacheMutex.Lock()
			state.DataCache[result.name] = result.data
			state.CacheMutex.Unlock()

			loadedCount++
			log.Printf("Loaded %s (%d/6)", result.name, loadedCount)

			// Update header with progress
			state.App.QueueUpdateDraw(func() {
				state.Header.SetText(fmt.Sprintf("[yellow]Loading data... (%d/6 complete)[-]", loadedCount))
			})

		case <-timeout:
			log.Printf("Data loading timed out after 2 minutes")
			state.App.QueueUpdateDraw(func() {
				state.Header.SetText("[red]Data loading timed out. Some data may be incomplete.[-]")
			})
			return
		}
	}

completed:
	log.Printf("All data preloaded successfully!")

	// Show dashboard by default
	state.App.QueueUpdateDraw(func() {
		state.Header.SetText("[green]AWS Cost Explorer - Dashboard[-] (Last updated: " + time.Now().Format("15:04:05") + ")")
		if dashboardData, exists := state.DataCache["Dashboard"]; exists {
			components.PopulateTable(state.MainTable, dashboardData)
		}
	})
}
