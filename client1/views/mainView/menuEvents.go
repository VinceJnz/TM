package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"strings"
	"time"
)

// MenuUserItem contains the basic user info for driving the display of the client menu
type MenuUser struct {
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Group     string `json:"group"`
	AdminFlag bool   `json:"admin_flag"`
}

// MenuItem contains data for a menu item
type MenuItem struct {
	UserID   int    `json:"user_id"`
	Resource string `json:"resource"`
}

// MenuItem contains a list of valid menu items to display
type MenuList []MenuItem

type UpdateMenu struct {
	MenuUser MenuUser
	MenuList MenuList
}

// *****************************************************************************
// Event handlers and event data types
// *****************************************************************************

// updateState is an event handler the updates the page status on the main page.
func (editor *View) updateState(event eventProcessor.Event) {
	state, ok := event.Data.(viewHelpers.ItemState)
	if !ok {
		log.Printf(debugTag+"Invalid data for event type: %s\n", event.Type)
		return
	}
	editor.ItemState.UpdateState(state)
}

// displayMessage is an event handler the displays a message on the main page.
func (v *View) displayMessage(event eventProcessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		log.Printf(debugTag+"Invalid data for event type: %s\n", event.Type)
		return
	}
	message = time.Now().Local().Format("15.04.05 02-01-2006") + `  "` + message + `"`
	v.ItemState.DisplayMessage(message)
}

// logoutComplete is an event handler the updates the logout status in the Navbar on the main page.
func (v *View) logoutComplete(event eventProcessor.Event) {
	v.elements.userDisplay.Set("innerHTML", "")
	v.resetMenu(event)
}

// loginComplete is an event handler the updates the login status in the Navbar on the main page.
func (v *View) loginComplete(event eventProcessor.Event) {
	username, ok := event.Data.(string)
	if !ok {
		log.Printf(debugTag+"Invalid data for event type: %s\n", event.Type)
		return
	}
	v.elements.userDisplay.Set("innerHTML", username)
	v.MenuProcess()
}

// resetMenu is an event handler resets the menu to display only the default menu items.
func (v *View) resetMenu(event eventProcessor.Event) {
	for _, o := range v.menuButtons {
		if !o.defaultDisplay {
			o.button.Get("style").Call("setProperty", "display", "none") // Hide menu item
		}
	}
}

// updateMenu is an event handler the updates the menu in the sidebar on the main page.
func (v *View) updateMenu(event eventProcessor.Event) {
	menuData, ok := event.Data.(UpdateMenu)
	if !ok {
		log.Printf(debugTag+"updateMenu()1 Invalid data for event type: %s\n", event.Type)
		return
	}
	if menuData.MenuUser.AdminFlag {
		for _, o := range v.menuButtons {
			o.button.Get("style").Call("removeProperty", "display")
		}
	} else {
		for _, o := range menuData.MenuList {
			val, ok := v.menuButtons[strings.ToLower(o.Resource)] // get current menu button
			log.Printf(debugTag+"updateMenu()2 Menu val=%+v, MenuItem=%+v,okay=%v\n", val, o, ok)
			if ok {
				val.button.Get("style").Call("removeProperty", "display")
			}
		}
	}
}
