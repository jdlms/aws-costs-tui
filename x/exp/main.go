package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppState struct {
	app        *tview.Application
	grid       *tview.Grid
	menu       *tview.List
	mainTable  *tview.Table
	header     *tview.TextView
	footer     *tview.TextView
	client     *costexplorer.Client
	loading    bool
	dataCache  map[string]CostData
	cacheMutex sync.RWMutex
}

type CostGroup struct {
	Name   string
	Amount float64
}

type CostData struct {
	Title string
	Rows  [][]string
}

func main() {
	// Setup logging to file to avoid interfering with TUI
	logFile, err := os.OpenFile("cost-explorer-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	client := costexplorer.NewFromConfig(cfg)

	appState := createApp(client)
	if err := appState.app.Run(); err != nil {
		panic(err)
	}
}

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

func createMainTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true).SetTitle("Cost Data")
	table.SetSelectable(true, false)
	return table
}

func createHeader() *tview.TextView {
	header := tview.NewTextView()
	header.SetBorder(true)
	header.SetText("AWS Cost Explorer - Loading...")
	header.SetTextAlign(tview.AlignCenter)
	header.SetDynamicColors(true)
	return header
}

func createFooter() *tview.TextView {
	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText("Press 'q' to quit | 'j/k' or ↑/↓ to navigate | Enter to select")
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetDynamicColors(true)
	return footer
}

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

func setupKeyBindings(app *tview.Application, state *AppState) {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		case 'j':
			// Move down in menu
			if app.GetFocus() == state.menu {
				currentIndex := state.menu.GetCurrentItem()
				itemCount := state.menu.GetItemCount()
				if currentIndex < itemCount-1 {
					state.menu.SetCurrentItem(currentIndex + 1)
				}
				return nil
			}
		case 'k':
			// Move up in menu
			if app.GetFocus() == state.menu {
				currentIndex := state.menu.GetCurrentItem()
				if currentIndex > 0 {
					state.menu.SetCurrentItem(currentIndex - 1)
				}
				return nil
			}
		}

		// Handle Enter key explicitly
		if event.Key() == tcell.KeyEnter {
			if app.GetFocus() == state.menu {
				// Get current selection and update content
				currentItem := state.menu.GetCurrentItem()
				menuItems := getMenuItems()
				if currentItem < len(menuItems) {
					log.Printf("Enter pressed - triggering selection for: %s", menuItems[currentItem])
					updateContent(state, menuItems[currentItem])
				}
				return nil
			}
		}

		return event
	})
}

func setupRosePineTheme() {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    tcell.NewRGBColor(35, 33, 54),    // base (#232136)
		ContrastBackgroundColor:     tcell.NewRGBColor(42, 39, 63),    // surface (#2a273f)
		MoreContrastBackgroundColor: tcell.NewRGBColor(57, 53, 82),    // overlay (#393552)
		BorderColor:                 tcell.NewRGBColor(110, 106, 134), // muted (#6e6a86)
		TitleColor:                  tcell.NewRGBColor(235, 188, 186), // rose (#ebbcba)
		GraphicsColor:               tcell.NewRGBColor(156, 207, 216), // foam (#9ccfd8)
		PrimaryTextColor:            tcell.NewRGBColor(224, 222, 244), // text (#e0def4)
		SecondaryTextColor:          tcell.NewRGBColor(144, 140, 170), // subtle (#908caa)
		TertiaryTextColor:           tcell.NewRGBColor(110, 106, 134), // muted (#6e6a86)
		InverseTextColor:            tcell.NewRGBColor(35, 33, 54),    // base (#232136)
		ContrastSecondaryTextColor:  tcell.NewRGBColor(224, 222, 244), // text (#e0def4)
	}
}

func createApp(client *costexplorer.Client) *AppState {
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

// Preload all data concurrently using the same pattern as main.go
func preloadAllData(state *AppState) {
	log.Printf("Starting concurrent data preload...")

	state.app.QueueUpdateDraw(func() {
		state.header.SetText("[yellow]Loading all cost data concurrently...[-]")
	})

	// Use WaitGroup to wait for all goroutines to complete - same as main.go
	var wg sync.WaitGroup

	// Use a channel to collect results
	results := make(chan struct {
		name string
		data CostData
		err  error
	}, 6)

	// Add a timeout for the entire operation
	timeout := time.After(2 * time.Minute)

	// Start all API calls concurrently - same pattern as main.go
	wg.Add(6)

	go func() {
		defer wg.Done()
		log.Printf("Fetching dashboard data...")
		data := getDashboardData(state.client)
		results <- struct {
			name string
			data CostData
			err  error
		}{"Dashboard", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching current month data...")
		data := getCurrentMonthData(state.client)
		results <- struct {
			name string
			data CostData
			err  error
		}{"Current Month", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching forecast data...")
		data := getForecastData(state.client)
		results <- struct {
			name string
			data CostData
			err  error
		}{"Forecast", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching service data...")
		data := getServiceData(state.client)
		results <- struct {
			name string
			data CostData
			err  error
		}{"By Service", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching region data...")
		data := getRegionData(state.client)
		results <- struct {
			name string
			data CostData
			err  error
		}{"By Region", data, nil}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Fetching usage type data...")
		data := getUsageTypeData(state.client)
		results <- struct {
			name string
			data CostData
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

			state.cacheMutex.Lock()
			state.dataCache[result.name] = result.data
			state.cacheMutex.Unlock()

			loadedCount++
			log.Printf("Loaded %s (%d/6)", result.name, loadedCount)

			// Update header with progress
			state.app.QueueUpdateDraw(func() {
				state.header.SetText(fmt.Sprintf("[yellow]Loading data... (%d/6 complete)[-]", loadedCount))
			})

		case <-timeout:
			log.Printf("Data loading timed out after 2 minutes")
			state.app.QueueUpdateDraw(func() {
				state.header.SetText("[red]Data loading timed out. Some data may be incomplete.[-]")
			})
			return
		}
	}

completed:
	log.Printf("All data preloaded successfully!")

	// Show dashboard by default
	state.app.QueueUpdateDraw(func() {
		state.header.SetText("[green]AWS Cost Explorer - Dashboard[-] (Last updated: " + time.Now().Format("15:04:05") + ")")
		if dashboardData, exists := state.dataCache["Dashboard"]; exists {
			populateTable(state.mainTable, dashboardData)
		}
	})
}

func updateContent(state *AppState, section string) {
	log.Printf("Starting updateContent for section: %s", section)

	// Check if data is cached
	state.cacheMutex.RLock()
	data, exists := state.dataCache[section]
	state.cacheMutex.RUnlock()

	log.Printf("Data exists for %s: %v", section, exists)

	if exists {
		// Use cached data - instant response!
		log.Printf("Using cached data for %s", section)

		// Update header first
		go func() {
			state.app.QueueUpdateDraw(func() {
				state.header.SetText(fmt.Sprintf("[green]AWS Cost Explorer - %s[-] (Cached data)", section))
			})
		}()

		// Then update table in a separate goroutine to prevent blocking
		go func() {
			state.app.QueueUpdateDraw(func() {
				log.Printf("About to populate table for %s", section)
				populateTable(state.mainTable, data)
				log.Printf("Finished populating table for %s", section)
			})
		}()

		log.Printf("Successfully queued UI updates for %s", section)
	} else {
		// Data not loaded yet, show loading message
		log.Printf("Data for %s not ready yet, showing loading message", section)

		emptyData := CostData{
			Title: fmt.Sprintf("%s - Loading...", section),
			Rows: [][]string{
				{"Status", "Message"},
				{"Loading", "Data is being fetched in the background..."},
			},
		}

		go func() {
			state.app.QueueUpdateDraw(func() {
				state.header.SetText(fmt.Sprintf("[yellow]%s data still loading...[-]", section))
			})
		}()

		go func() {
			state.app.QueueUpdateDraw(func() {
				populateTable(state.mainTable, emptyData)
			})
		}()
	}
}

func populateTable(table *tview.Table, data CostData) {
	log.Printf("Starting to populate table with title: %s, rows: %d", data.Title, len(data.Rows))

	// Ensure we have a valid table
	if table == nil {
		log.Printf("ERROR: table is nil!")
		return
	}

	table.Clear()
	table.SetTitle(data.Title)
	log.Printf("Table cleared and title set to: %s", data.Title)

	if len(data.Rows) == 0 {
		table.SetCell(0, 0, tview.NewTableCell("No data available").
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
		log.Printf("Table populated with no data message")
		return
	}

	// Add header row if present
	if len(data.Rows) > 0 && len(data.Rows[0]) > 0 {
		log.Printf("Adding header row with %d columns", len(data.Rows[0]))
		for col, cell := range data.Rows[0] {
			table.SetCell(0, col, tview.NewTableCell("[yellow::b]"+cell+"[-::-]").
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
		}
	}

	// Add data rows
	for row := 1; row < len(data.Rows); row++ {
		if len(data.Rows[row]) == 0 {
			log.Printf("WARNING: Row %d is empty, skipping", row)
			continue
		}

		for col, cell := range data.Rows[row] {
			color := "[white]"
			// Color code amounts (assuming last column is usually the amount)
			if col == len(data.Rows[row])-1 && strings.HasPrefix(cell, "$") {
				if strings.Contains(cell, "$0.00") {
					color = "[gray]"
				} else {
					amount := parseAmount(cell)
					if amount > 100 {
						color = "[red]"
					} else if amount > 10 {
						color = "[orange]"
					} else {
						color = "[green]"
					}
				}
			}

			table.SetCell(row, col, tview.NewTableCell(color+cell+"[-]").
				SetAlign(tview.AlignLeft).
				SetSelectable(row != 0))
		}
	}

	log.Printf("Table populated successfully with %d rows", len(data.Rows))
}

func parseAmount(amountStr string) float64 {
	// Remove $ and commas, then parse
	cleaned := strings.ReplaceAll(strings.TrimPrefix(amountStr, "$"), ",", "")
	amount, _ := strconv.ParseFloat(cleaned, 64)
	return amount
}

// Data fetching functions using the exact patterns from main.go
func getDashboardData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	// Get first day of current month
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// Get first day of next month
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost", "UnblendedCost", "NetUnblendedCost"},
	})

	rows := [][]string{
		{"Metric", "Amount", "Period"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Dashboard Overview", Rows: rows}
	}

	for _, resultByTime := range result.ResultsByTime {
		period := fmt.Sprintf("%s to %s", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End)

		if blendedCost, exists := resultByTime.Total["BlendedCost"]; exists {
			rows = append(rows, []string{"Total Blended Cost", formatCost(blendedCost.Amount), period})
		}
		if unblendedCost, exists := resultByTime.Total["UnblendedCost"]; exists {
			rows = append(rows, []string{"Total Unblended Cost", formatCost(unblendedCost.Amount), period})
		}
		if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
			rows = append(rows, []string{"Total Net Cost", formatCost(netCost.Amount), period})
		}
	}

	return CostData{Title: "💸 Dashboard Overview", Rows: rows}
}

func getForecastData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	// Forecast for next month - exact logic from main.go
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if nextMonthStart.Month() == 1 { // Handle year rollover
		nextMonthStart = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	}
	nextMonthEnd := nextMonthStart.AddDate(0, 1, 0)

	startDate := nextMonthStart.Format("2006-01-02")
	endDate := nextMonthEnd.Format("2006-01-02")

	forecast, err := client.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metric:      types.MetricBlendedCost,
	})

	rows := [][]string{
		{"Forecast Type", "Amount", "Period"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Cost Forecast", Rows: rows}
	}

	period := fmt.Sprintf("%s to %s", startDate, endDate)
	rows = append(rows, []string{"Predicted Total Cost", formatCost(forecast.Total.Amount), period})

	if forecast.ForecastResultsByTime != nil {
		for _, forecastResult := range forecast.ForecastResultsByTime {
			forecastPeriod := fmt.Sprintf("%s to %s", *forecastResult.TimePeriod.Start, *forecastResult.TimePeriod.End)
			rows = append(rows, []string{"Mean Estimate", formatCost(forecastResult.MeanValue), forecastPeriod})
		}
	}

	return CostData{Title: "🔮 Cost Forecast", Rows: rows}
}

func getServiceData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set time range for last 30 days - exact logic from main.go
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Format dates as YYYY-MM-DD
	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get cost and usage data, grouped by service - exact API call from main.go
	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"SERVICE"}[0],
			},
		},
	})

	rows := [][]string{
		{"Service", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Costs by Service", Rows: rows}
	}

	var totalCost float64
	serviceMap := make(map[string]float64)

	// Aggregate costs across all time periods by service name
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				serviceName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						serviceMap[serviceName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	// Convert map to slice for sorting
	var costGroups []CostGroup
	for serviceName, amount := range serviceMap {
		costGroups = append(costGroups, CostGroup{
			Name:   serviceName,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	for _, group := range costGroups {
		percentage := (group.Amount / totalCost) * 100
		amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
		rows = append(rows, []string{
			group.Name,
			formatCost(&amountStr),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	return CostData{Title: "🛠️ Last 30 Days Costs by Service", Rows: rows}
}

func getRegionData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get costs by region - exact API call from main.go
	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"REGION"}[0],
			},
		},
	})

	rows := [][]string{
		{"Region", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Costs by Region", Rows: rows}
	}

	var totalCost float64
	regionMap := make(map[string]float64)

	// Aggregate costs across all time periods by region name
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				regionName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						regionMap[regionName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	// Convert map to slice for sorting
	var costGroups []CostGroup
	for regionName, amount := range regionMap {
		costGroups = append(costGroups, CostGroup{
			Name:   regionName,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	for _, group := range costGroups {
		percentage := (group.Amount / totalCost) * 100
		amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
		rows = append(rows, []string{
			group.Name,
			formatCost(&amountStr),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	return CostData{Title: "🌍 Costs by Region", Rows: rows}
}

func getUsageTypeData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get costs by usage type - exact API call from main.go
	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"USAGE_TYPE"}[0],
			},
		},
	})

	rows := [][]string{
		{"Usage Type", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Costs by Usage Type", Rows: rows}
	}

	var totalCost float64
	usageTypeMap := make(map[string]float64)

	// Aggregate costs across all time periods by usage type
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				usageTypeName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						usageTypeMap[usageTypeName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	// Convert map to slice for sorting
	var costGroups []CostGroup
	for usageTypeName, amount := range usageTypeMap {
		costGroups = append(costGroups, CostGroup{
			Name:   usageTypeName,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	// Take only top 10
	if len(costGroups) > 10 {
		costGroups = costGroups[:10]
		// Recalculate total for percentage calculation
		totalCost = 0
		for _, group := range costGroups {
			totalCost += group.Amount
		}
	}

	for _, group := range costGroups {
		percentage := (group.Amount / totalCost) * 100
		amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
		rows = append(rows, []string{
			group.Name,
			formatCost(&amountStr),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	return CostData{Title: "📊 Top 10 Usage Types", Rows: rows}
}

func getCurrentMonthData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	// Get first day of current month - exact logic from main.go
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// Get first day of next month
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost", "UnblendedCost", "NetUnblendedCost"},
	})

	rows := [][]string{
		{"Period", "Metric", "Amount"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Current Month Breakdown", Rows: rows}
	}

	for _, resultByTime := range result.ResultsByTime {
		period := fmt.Sprintf("%s to %s", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End)

		if blendedCost, exists := resultByTime.Total["BlendedCost"]; exists {
			rows = append(rows, []string{period, "Total Blended Cost", formatCost(blendedCost.Amount)})
		}
		if unblendedCost, exists := resultByTime.Total["UnblendedCost"]; exists {
			rows = append(rows, []string{period, "Total Unblended Cost", formatCost(unblendedCost.Amount)})
		}
		if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
			rows = append(rows, []string{period, "Total Net Cost", formatCost(netCost.Amount)})
		}
	}

	return CostData{Title: fmt.Sprintf("📅 Current Month Costs (%s)", now.Format("January 2006")), Rows: rows}
}

// Helper functions from your original code
func formatCost(amountStr *string) string {
	if amountStr == nil {
		return "$0.00"
	}

	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil {
		return "$0.00"
	}

	rounded := math.Ceil(amount*100) / 100
	return fmt.Sprintf("$%.2f", rounded)
}

func parseAmountString(amountStr *string) float64 {
	if amountStr == nil {
		return 0.0
	}
	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil {
		return 0.0
	}
	return amount
}

func sortCostGroups(groups []types.Group) []CostGroup {
	var costGroups []CostGroup

	for _, group := range groups {
		name := "Unknown"
		if len(group.Keys) > 0 {
			name = group.Keys[0]
		}

		amount := 0.0
		if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
			if parsed, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil {
				amount = parsed
			}
		}

		costGroups = append(costGroups, CostGroup{
			Name:   name,
			Amount: amount,
		})
	}

	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	return costGroups
}
