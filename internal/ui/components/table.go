package components

import (
	"log"
	"strings"

	"github.com/rivo/tview"
)

func createMainTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true).SetTitle("Cost Data")
	table.SetSelectable(true, false)
	return table
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
