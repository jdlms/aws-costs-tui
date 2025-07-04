// cost_data.go - common cost data utilities and helpers
package aws

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

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

func parseAmount(amountStr string) float64 {
	// Remove $ and commas, then parse
	cleaned := strings.ReplaceAll(strings.TrimPrefix(amountStr, "$"), ",", "")
	amount, _ := strconv.ParseFloat(cleaned, 64)
	return amount
}

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
