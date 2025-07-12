package app

import (
	"log"

	"cost-explorer/internal/types"

	"github.com/gdamore/tcell/v2"
)

// GetMenuItems returns the list of menu items
func GetMenuItems() []string {
	return []string{
		"Dashboard",
		"By Service",
		"By Usage Type",
		"By Region",
	}
}

// SetupKeyBindings configures keyboard input handling
func SetupKeyBindings(state *types.AppState, updateContentFunc func(*types.AppState, string)) {
	state.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentFocus := state.App.GetFocus()

		switch event.Rune() {
		case 'q':
			state.App.Stop()
			return nil
		case 'j':
			// Move down in menu or table
			if currentFocus == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				itemCount := state.Menu.GetItemCount()
				if currentIndex < itemCount-1 {
					state.Menu.SetCurrentItem(currentIndex + 1)
				}
				return nil
			} else if currentFocus == state.MainTable {
				// Move down in table
				row, col := state.MainTable.GetSelection()
				rowCount := state.MainTable.GetRowCount()
				if row < rowCount-1 {
					state.MainTable.Select(row+1, col)
				}
				return nil
			}
		case 'k':
			// Move up in menu or table
			if currentFocus == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				if currentIndex > 0 {
					state.Menu.SetCurrentItem(currentIndex - 1)
				}
				return nil
			} else if currentFocus == state.MainTable {
				// Move up in table
				row, col := state.MainTable.GetSelection()
				if row > 1 { // Don't go above header row (row 0)
					state.MainTable.Select(row-1, col)
				}
				return nil
			}
		}

		// Handle special keys
		switch event.Key() {
		case tcell.KeyUp:
			// Arrow up - same as 'k'
			if currentFocus == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				if currentIndex > 0 {
					state.Menu.SetCurrentItem(currentIndex - 1)
				}
				return nil
			} else if currentFocus == state.MainTable {
				row, col := state.MainTable.GetSelection()
				if row > 1 { // Don't go above header row (row 0)
					state.MainTable.Select(row-1, col)
				}
				return nil
			}
		case tcell.KeyDown:
			// Arrow down - same as 'j'
			if currentFocus == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				itemCount := state.Menu.GetItemCount()
				if currentIndex < itemCount-1 {
					state.Menu.SetCurrentItem(currentIndex + 1)
				}
				return nil
			} else if currentFocus == state.MainTable {
				row, col := state.MainTable.GetSelection()
				rowCount := state.MainTable.GetRowCount()
				if row < rowCount-1 {
					state.MainTable.Select(row+1, col)
				}
				return nil
			}
		case tcell.KeyTab:
			// Switch focus back to menu from table
			if currentFocus == state.MainTable {
				// Clear table selection by moving to header row
				state.MainTable.Select(0, 0)
				state.App.SetFocus(state.Menu)
				return nil
			}
		case tcell.KeyEnter:
			if currentFocus == state.Menu {
				// Get current selection and update content, then switch to table
				currentItem := state.Menu.GetCurrentItem()
				menuItems := GetMenuItems()
				if currentItem < len(menuItems) {
					log.Printf("Enter pressed - triggering selection for: %s", menuItems[currentItem])
					updateContentFunc(state, menuItems[currentItem])
					// Switch focus to table after loading content
					state.App.SetFocus(state.MainTable)
				}
				return nil
			}
		case tcell.KeyPgDn:
			// Page down in table
			if currentFocus == state.MainTable {
				row, col := state.MainTable.GetSelection()
				rowCount := state.MainTable.GetRowCount()
				newRow := row + 10 // Move 10 rows down
				if newRow >= rowCount {
					newRow = rowCount - 1
				}
				if newRow < 1 { // Don't go above first data row
					newRow = 1
				}
				state.MainTable.Select(newRow, col)
				return nil
			}
		case tcell.KeyPgUp:
			// Page up in table
			if currentFocus == state.MainTable {
				row, col := state.MainTable.GetSelection()
				newRow := row - 10 // Move 10 rows up
				if newRow < 1 {    // Don't go above first data row
					newRow = 1
				}
				state.MainTable.Select(newRow, col)
				return nil
			}
		}

		return event
	})
}
