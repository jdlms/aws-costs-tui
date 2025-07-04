package ui

import (
	"log"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

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
