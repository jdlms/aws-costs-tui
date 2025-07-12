package ui

import (
	"log"
	"strconv"
	"strings"

	"cost-explorer/internal/types"

	"github.com/rivo/tview"
)

// PopulateTable fills the table with cost data
func PopulateTable(table *tview.Table, data types.CostData) {
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

	// Identify top 3 costs for each month column (for service data highlighting)
	topCostsByColumn := make(map[int]map[int]bool) // column -> row -> isTop3
	if strings.Contains(data.Title, "Services") && len(data.Rows) > 1 {
		topCostsByColumn = findTopCostsPerColumn(data.Rows)
	}

	// Add data rows
	for row := 1; row < len(data.Rows); row++ {
		if len(data.Rows[row]) == 0 {
			log.Printf("WARNING: Row %d is empty, skipping", row)
			continue
		}

		for col, cell := range data.Rows[row] {
			color := "[white]"

			// Check if this cell should be highlighted as top 3 cost
			if topCostsByColumn[col] != nil && topCostsByColumn[col][row] {
				color = "[yellow]"
			} else if strings.HasPrefix(cell, "$") {
				color = "[-]" // Default color for dollar amounts
			}

			table.SetCell(row, col, tview.NewTableCell(color+cell+"[-]").
				SetAlign(tview.AlignLeft).
				SetSelectable(true))
		}
	}

	log.Printf("Table populated successfully with %d rows", len(data.Rows))
}

// parseAmount extracts the numeric value from a cost string
func parseAmount(amountStr string) float64 {
	// Remove $ and commas, then parse
	cleaned := strings.ReplaceAll(strings.TrimPrefix(amountStr, "$"), ",", "")
	amount, _ := strconv.ParseFloat(cleaned, 64)
	return amount
}

// findTopCostsPerColumn identifies the top 3 costs in each month column for highlighting
func findTopCostsPerColumn(rows [][]string) map[int]map[int]bool {
	result := make(map[int]map[int]bool)

	if len(rows) < 2 {
		return result
	}

	// Skip first column (service names) and process cost columns
	for col := 1; col < len(rows[0]); col++ {
		// Collect all costs for this column with their row indices
		type costRow struct {
			amount float64
			row    int
		}

		var costs []costRow
		for row := 1; row < len(rows); row++ {
			if col < len(rows[row]) {
				amount := parseAmount(rows[row][col])
				if amount > 0 {
					costs = append(costs, costRow{amount: amount, row: row})
				}
			}
		}

		// Sort by amount descending
		for i := 0; i < len(costs)-1; i++ {
			for j := i + 1; j < len(costs); j++ {
				if costs[i].amount < costs[j].amount {
					costs[i], costs[j] = costs[j], costs[i]
				}
			}
		}

		// Mark top 3 costs for this column
		result[col] = make(map[int]bool)
		for i := 0; i < len(costs) && i < 3; i++ {
			result[col][costs[i].row] = true
		}
	}

	return result
}
