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
	UserID   int    `json:"user_id"`
	Resource string `json:"resource"`
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
	for _, section := range v.menuSections {
		section.container.Get("style").Call("setProperty", "display", "none")
		section.content.Get("style").Call("setProperty", "display", "none")
	}
}

// updateMenu is an event handler that updates the menu in the sidebar on the main page.
func (v *View) updateMenu(event eventProcessor.Event) {
	menuData, ok := event.Data.(UpdateMenu)
	if !ok {
		log.Printf(debugTag+"updateMenu()1 Invalid data for event type: %s, source %s\n", event.Type, event.DebugTag)
		return
	}

	v.resetMenu(event)

	userRole := effectiveMenuUserRole(menuData.MenuUser)

	allowedResources := map[string]bool{}
	for _, o := range menuData.MenuList {
		allowedResources[strings.ToLower(o.Resource)] = true
	}

	visibleSections := map[string]bool{}
	for key, item := range v.menuButtons {
		if item.defaultDisplay {
			continue
		}
		if !canDisplayByRole(userRole, item.requiredRole) {
			continue
		}
		if shouldFilterByResource(userRole) && !allowedResources[key] {
			continue
		}
		item.button.Get("style").Call("removeProperty", "display")
		if item.section != menuSectionRoot {
			visibleSections[item.section] = true
		}
	}

	for sectionName, visible := range visibleSections {
		if !visible {
			continue
		}
		section, ok := v.menuSections[sectionName]
		if !ok {
			continue
		}
		section.container.Get("style").Call("removeProperty", "display")
	}
}

func canDisplayByRole(userRole, requiredRole string) bool {
	roleRank := map[string]int{
		roleUser:     1,
		roleAdmin:    2,
		roleSysadmin: 3,
	}
	userValue, ok := roleRank[normalizeRole(userRole)]
	if !ok {
		return false
	}
	requiredValue, ok := roleRank[normalizeRole(requiredRole)]
	if !ok {
		return false
	}
	return userValue >= requiredValue
}

func effectiveMenuUserRole(menuUser appCore.User) string {
	return normalizeRole(menuUser.Role)
}

func shouldFilterByResource(userRole string) bool {
	normalized := normalizeRole(userRole)
	return normalized == roleUser
}
