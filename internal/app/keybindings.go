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
		"Current Month",
		"Forecast",
		"By Service",
		"By Region",
		"By Usage Type",
	}
}

// SetupKeyBindings configures keyboard input handling
func SetupKeyBindings(state *types.AppState, updateContentFunc func(*types.AppState, string)) {
	state.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			state.App.Stop()
			return nil
		case 'j':
			// Move down in menu
			if state.App.GetFocus() == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				itemCount := state.Menu.GetItemCount()
				if currentIndex < itemCount-1 {
					state.Menu.SetCurrentItem(currentIndex + 1)
				}
				return nil
			}
		case 'k':
			// Move up in menu
			if state.App.GetFocus() == state.Menu {
				currentIndex := state.Menu.GetCurrentItem()
				if currentIndex > 0 {
					state.Menu.SetCurrentItem(currentIndex - 1)
				}
				return nil
			}
		}

		// Handle Enter key explicitly
		if event.Key() == tcell.KeyEnter {
			if state.App.GetFocus() == state.Menu {
				// Get current selection and update content
				currentItem := state.Menu.GetCurrentItem()
				menuItems := GetMenuItems()
				if currentItem < len(menuItems) {
					log.Printf("Enter pressed - triggering selection for: %s", menuItems[currentItem])
					updateContentFunc(state, menuItems[currentItem])
				}
				return nil
			}
		}

		return event
	})
}
