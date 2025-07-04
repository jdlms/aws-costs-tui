package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppState struct {
	app  *tview.Application
	grid *tview.Grid
	menu *tview.List
	main *tview.TextView
	// sidebar *tview.Table
	header *tview.TextView
	footer *tview.TextView
}

// Dummy data functions
func getMenuItems() []string {
	return []string{"Dashboard", "Users", "Settings", "Reports", "Help"}
}

func getMainContent(section string) string {
	content := map[string]string{
		"Dashboard": "Welcome to the dashboard!\n\nSystem Status: Online\nActive Users: 42\nLast Update: " + time.Now().Format("15:04:05"),
		"Users":     "User Management\n\nTotal Users: 1,234\nActive: 987\nInactive: 247\n\nRecent Activity:\n- John logged in\n- Jane updated profile\n- Bob logged out",
		"Settings":  "Application Settings\n\nTheme: Dark\nLanguage: English\nNotifications: Enabled\nAuto-save: Every 5 minutes",
		"Reports":   "System Reports\n\nDaily Active Users: 156\nError Rate: 0.02%\nResponse Time: 45ms\nUptime: 99.9%",
		"Help":      "Help & Documentation\n\nUser Guide: Available\nAPI Docs: /api/docs\nSupport: help@company.com\nVersion: 1.2.3",
	}
	if val, exists := content[section]; exists {
		return val
	}
	return "Select a menu item to view content"
}

// func getSidebarData() [][]string {
// 	return [][]string{
// 		{"Metric", "Value", "Status"},
// 		{"CPU", "45%", "OK"},
// 		{"Memory", "67%", "Warning"},
// 		{"Disk", "23%", "OK"},
// 		{"Network", "12MB/s", "OK"},
// 		{"Errors", "3", "Warning"},
// 		{"Uptime", "5d 12h", "OK"},
// 	}
// }

func getHeaderText() string {
	return fmt.Sprintf("TUI Dashboard - %s", time.Now().Format("2006-01-02 15:04:05"))
}

func getFooterText() string {
	return "Press 'q' to quit | Tab to navigate | Enter to select"
}

func createMenu(onSelect func(string)) *tview.List {
	menu := tview.NewList()
	menu.SetBorder(true).SetTitle("Navigation")

	for _, item := range getMenuItems() {
		menu.AddItem(item, "", 0, nil)
	}

	menu.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		onSelect(mainText)
	})

	return menu
}

func createMainContent() *tview.TextView {
	textView := tview.NewTextView()
	textView.SetBorder(true).SetTitle("Main Content")
	textView.SetText(getMainContent("Dashboard"))
	textView.SetScrollable(true).SetWrap(true)
	return textView
}

// func createSidebar() *tview.Table {
// 	table := tview.NewTable()
// 	table.SetBorder(true).SetTitle("System Status")

// 	data := getSidebarData()
// 	for row, rowData := range data {
// 		for col, cellData := range rowData {
// 			color := "[white]"
// 			if row == 0 {
// 				color = "[yellow]" // Header row
// 			} else if col == 2 { // Status column
// 				switch cellData {
// 				case "OK":
// 					color = "[green]"
// 				case "Warning":
// 					color = "[orange]"
// 				case "Error":
// 					color = "[red]"
// 				}
// 			}

// 			table.SetCell(row, col, tview.NewTableCell(color+cellData).
// 				SetAlign(tview.AlignCenter).
// 				SetSelectable(row != 0))
// 		}
// 	}

// 	return table
// }

func createHeader() *tview.TextView {
	header := tview.NewTextView()
	header.SetBorder(true)
	header.SetText(getHeaderText())
	header.SetTextAlign(tview.AlignCenter)
	return header
}

func createFooter() *tview.TextView {
	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText(getFooterText())
	footer.SetTextAlign(tview.AlignCenter)
	return footer
}

func updateMainContent(main *tview.TextView, section string) {
	main.SetText(getMainContent(section))
}

func updateHeader(header *tview.TextView) {
	header.SetText(getHeaderText())
}

func setupGrid(state *AppState) *tview.Grid {
	grid := tview.NewGrid().
		SetRows(2, 0).
		SetColumns(25, 0).
		SetBorders(false) // We handle borders on individual components

	// Static items
	grid.AddItem(state.header, 0, 0, 1, 3, 0, 0, false)
	grid.AddItem(state.footer, 2, 0, 1, 3, 0, 0, false)

	// Main layout
	grid.AddItem(state.menu, 1, 0, 1, 1, 0, 80, true)
	grid.AddItem(state.main, 1, 1, 1, 1, 0, 80, false)

	return grid
}

func setupKeyBindings(app *tview.Application) {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		}
		return event
	})
}

func startPeriodicUpdates(state *AppState) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			state.app.QueueUpdateDraw(func() {
				updateHeader(state.header)
			})
		}
	}()
}

func createApp() *AppState {
	state := &AppState{}

	// Create components with data
	state.header = createHeader()
	state.footer = createFooter()
	state.main = createMainContent()
	// state.sidebar = createSidebar()

	// Menu with callback to update main content
	state.menu = createMenu(func(selection string) {
		updateMainContent(state.main, selection)
	})

	// Setup grid
	state.grid = setupGrid(state)

	// Create application
	state.app = tview.NewApplication().
		SetRoot(state.grid, true).
		SetFocus(state.menu)

	// Setup key bindings
	setupKeyBindings(state.app)

	// Start periodic updates
	startPeriodicUpdates(state)

	return state
}

func main() {
	appState := createApp()

	if err := appState.app.Run(); err != nil {
		panic(err)
	}
}
