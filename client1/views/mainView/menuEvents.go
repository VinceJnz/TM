package mainView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"strings"
)

// MenuItem contains data for a menu item
type MenuItem struct {
	UserID    int    `json:"user_id"`
	Resource  string `json:"resource"`
	AdminFlag bool   `json:"admin_flag"`
}

// MenuItem contains a list of valid menu items to display
type MenuList []MenuItem

type UpdateMenu struct {
	MenuUser appCore.User
	MenuList MenuList
}

// *****************************************************************************
// Event handlers and event data types
// *****************************************************************************

// updateState is an event handler the updates the page status on the main page.
func (editor *View) updateStatus(event eventProcessor.Event) {
	state, ok := event.Data.(viewHelpers.ItemState)
	if !ok {
		log.Printf(debugTag+"updateStatus()1 Invalid data for event type: %s, source %s\n", event.Type, event.DebugTag)
		return
	}
	editor.ItemState.UpdateStatus(state, event.DebugTag)
}

// displayMessage is an event handler the displays a message on the main page.
func (v *View) displayMessage(event eventProcessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		log.Printf(debugTag+"displayMessage()1 Invalid data for event type: %s, source %s\n", event.Type, event.DebugTag)
		return
	}
	//message = time.Now().Local().Format("15.04.05 02-01-2006") + `  "` + message + `"`
	v.ItemState.DisplayMessage(message)
}

// logoutComplete is an event handler the updates the logout status in the Navbar on the main page.
func (v *View) logoutComplete(event eventProcessor.Event) {
	v.elements.userDisplay.Set("innerHTML", "")
	v.resetMenu(event)
	v.ResetViewItems()
}

// loginComplete is an event handler the updates the login status in the Navbar on the main page.
func (v *View) loginComplete(event eventProcessor.Event) {
	username, ok := event.Data.(string)
	if !ok {
		log.Printf(debugTag+"loginComplete()1 Invalid data for event type: %s, source %s\n", event.Type, event.DebugTag)
		return
	}
	v.elements.userDisplay.Set("innerHTML", username)
	v.MenuProcess()
}

// resetMenu is an event handler that resets the menu to display only the default menu items.
func (v *View) resetMenu(event eventProcessor.Event) {
	for _, o := range v.menuButtons {
		if !o.defaultDisplay {
			o.button.Get("style").Call("setProperty", "display", "none") // Hide menu item - set property "display: none;"
		}
	}
}

// updateMenu is an event handler that updates the menu in the sidebar on the main page.
func (v *View) updateMenu(event eventProcessor.Event) {
	menuData, ok := event.Data.(UpdateMenu)
	if !ok {
		log.Printf(debugTag+"updateMenu()1 Invalid data for event type: %s, source %s\n", event.Type, event.DebugTag)
		return
	}
	if menuData.MenuUser.AdminFlag {
		for _, o := range v.menuButtons {
			o.button.Get("style").Call("removeProperty", "display") // Remove property "display: none;" causes the menu button to be displayed
		}
	} else {
		for _, o := range menuData.MenuList { // Iterate through the menu list from the server and hide/display buttons as necessary.
			val, ok := v.menuButtons[strings.ToLower(o.Resource)]
			if ok {
				if !val.adminOnly {
					val.button.Get("style").Call("removeProperty", "display") // Remove property "display: none;" causes the menu button to be displayed
				}
			}
		}
	}
}
