package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"strings"
)

const homeMenuTitle = "Home"

func (v *View) setTopAuthAction(isLoggedIn bool) {
	if loginBtn, ok := v.menuTitles["Login"]; ok {
		if isLoggedIn {
			loginBtn.Get("style").Call("setProperty", "display", "none")
		} else {
			loginBtn.Get("style").Call("removeProperty", "display")
		}
	}

	if logoutBtn, ok := v.menuTitles["Logout"]; ok {
		if isLoggedIn {
			logoutBtn.Get("style").Call("removeProperty", "display")
		} else {
			logoutBtn.Get("style").Call("setProperty", "display", "none")
		}
	}

	if !v.elements.myProfileBtn.IsUndefined() && !v.elements.myProfileBtn.IsNull() {
		if isLoggedIn {
			v.elements.myProfileBtn.Get("style").Call("removeProperty", "display")
		} else {
			v.elements.myProfileBtn.Get("style").Call("setProperty", "display", "none")
		}
	}
}

// MenuItem contains data for a menu item
type MenuItem struct {
	UserID      int    `json:"user_id"`
	Resource    string `json:"resource"`
	AccessLevel string `json:"access_level"`
	AccessScope string `json:"access_scope"`
}

// MenuItem contains a list of valid menu items to display
type MenuList []MenuItem

func (v *View) logInvalidEventData(handlerName string, event eventProcessor.Event) {
	log.Printf(debugTag+"%s Invalid data for event type: %s, source %s\n", handlerName, event.Type, event.DebugTag)
}

func (v *View) hideMenuButton(item buttonElement) {
	item.button.Get("style").Call("setProperty", "display", "none")
}

func (v *View) showMenuButton(item buttonElement) {
	item.button.Get("style").Call("removeProperty", "display")
}

func (v *View) collapseMenuSection(section sectionElement) {
	section.container.Get("style").Call("setProperty", "display", "none")
	section.content.Get("style").Call("setProperty", "display", "none")
	section.header.Set("aria-expanded", "false")
	section.header.Get("classList").Call("remove", "btn-active")
}

func (v *View) showMenuSectionCollapsed(section sectionElement) {
	section.container.Get("style").Call("removeProperty", "display")
	section.header.Set("aria-expanded", "false")
	section.header.Get("classList").Call("remove", "btn-active")
}

// Event handlers

// updateState is an event handler the updates the page status on the main page.
func (editor *View) updateStatus(event eventProcessor.Event) {
	state, ok := event.Data.(viewHelpers.ItemState)
	if !ok {
		editor.logInvalidEventData("updateStatus()1", event)
		return
	}
	editor.ItemState.UpdateStatus(state, event.DebugTag)
}

// displayMessage is an event handler the displays a message on the main page.
func (v *View) displayMessage(event eventProcessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		v.logInvalidEventData("displayMessage()1", event)
		return
	}
	v.ItemState.DisplayMessage(message)
}

// logoutComplete is an event handler the updates the logout status in the Navbar on the main page.
func (v *View) logoutComplete(event eventProcessor.Event) {
	v.elements.userDisplay.Set("innerHTML", "")
	v.setTopAuthAction(false)
	v.profileReturnTitle = ""
	v.resetMenu()
	v.ResetViewItems()
	v.menuOnClick(homeMenuTitle, true, nil)()
}

// loginComplete is an event handler the updates the login status in the Navbar on the main page.
func (v *View) loginComplete(event eventProcessor.Event) {
	username, ok := event.Data.(string)
	if !ok {
		v.logInvalidEventData("loginComplete()1", event)
		return
	}
	v.elements.userDisplay.Set("innerHTML", username)
	v.setTopAuthAction(true)
	v.MenuProcess()
	v.menuOnClick(homeMenuTitle, true, nil)()
}

func (v *View) resetMenuEvent(event eventProcessor.Event) {
	v.resetMenu()
}

// resetMenu is an event handler that resets the menu to display only the default menu items.
func (v *View) resetMenu() {
	for key, o := range v.menuButtons {
		if key == "login" || key == "logout" {
			o.button.Get("classList").Call("remove", "menu-item-active")
			continue
		}
		if !o.defaultDisplay {
			v.hideMenuButton(o)
		}
		o.button.Get("classList").Call("remove", "menu-item-active")
	}
	for _, section := range v.menuSections {
		v.collapseMenuSection(section)
	}
}

// updateMenu is an event handler that updates the menu in the sidebar on the main page.
func (v *View) updateMenu(event eventProcessor.Event) {
	menuList, ok := event.Data.(MenuList)
	if !ok {
		v.logInvalidEventData("updateMenu()1", event)
		return
	}

	v.resetMenu()

	allowedResources := map[string]struct{}{}
	for _, o := range menuList {
		allowedResources[strings.ToLower(o.Resource)] = struct{}{}
	}

	visibleSections := map[string]struct{}{}
	for key, item := range v.menuButtons {
		if item.defaultDisplay {
			continue
		}
		if _, allowed := allowedResources[key]; !allowed {
			continue
		}
		v.showMenuButton(item)
		if item.section != menuSectionRoot {
			visibleSections[item.section] = struct{}{}
		}
	}

	for sectionName := range visibleSections {
		section, ok := v.menuSections[sectionName]
		if !ok {
			continue
		}
		v.showMenuSectionCollapsed(section)
	}
}
