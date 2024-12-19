package mainView

import (
	"client1/v2/views/utils/viewHelpers"
	"time"
)

// Example
// DisplayStatus provides an object to send data to the even handler. This is an examle, it is not used.
type DisplayStatus struct {
	Message string
}

// ************************************************************************************
// ************************************************************************************
// ************************************************************************************
// Event processing
// ************************************************************************************

// logoutComplete is an event handler the updates the logout status in the Navbar on the main page.
func (v *View) logoutComplete2() {
	v.elements.userDisplay.Set("innerHTML", "")
	v.resetMenu2()
}

// loginComplete is an event handler the updates the login status in the Navbar on the main page.
func (v *View) loginComplete2(username string) {
	// Update menu display??? on fetch success????
	v.elements.userDisplay.Set("innerHTML", username)
	v.MenuProcess()
}

// resetMenu is an event handler resets the menu to display only the default menu items.
func (v *View) resetMenu2() {
	for _, o := range v.menuButtons {
		if !o.defaultDisplay {
			o.button.Get("style").Call("setProperty", "display", "none") // Hide menu item
		}
	}
}

// updateStatus is an event handler the updates the menu in the sidebar on the main page.
func (v *View) updateMenu2(menuData UpdateMenu) {
	// Update menu display??? on fetch success????
	if menuData.MenuUser.AdminFlag {
		for _, o := range v.menuButtons {
			o.button.Get("style").Call("removeProperty", "display")
		}
	} else {
		for _, o := range menuData.MenuList {
			val, ok := v.menuButtons["/"+o.Resource] // get current menu button
			if ok {
				val.button.Get("style").Call("removeProperty", "display")
			}
		}
	}
}

func (editor *View) OnAction(action interface{}) {
	switch a := action.(type) {
	case *SetStatus:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.ItemState.UpdateStatus(a.ItemState)
	case *DisplayMessage:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.ItemState.DisplayMessage(a.Message)
	case *SetMenu:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.updateMenu2(a.MenuData)
	case *ResetMenu:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.resetMenu2()
	case *LoginComplete:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.loginComplete2(a.Username)
	case **LogoutComplete:
		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
		editor.logoutComplete2()
	default:
		//log.Printf("%v %v %T %+v", debugTag+"Store.OnAction()Default - invalid action type (action should be a pointer e.g. &struct.Action) ", "a =", a, a)
		return // don't fire listeners
	}
	//Listeners.Fire()
}

// Actions
// SetStatus is an event handler the updates the page status on the main page.
type SetStatus struct {
	Time      time.Time
	ItemState viewHelpers.ItemState
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// SetStatus is an event handler the updates the page status on the main page.
type DisplayMessage struct {
	Time    time.Time
	Message string
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// SetStatus is an event handler the updates the page status on the main page.
type ResetMenu struct {
	Time    time.Time
	Message string
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// SetStatus is an event handler the updates the page status on the main page.
type SetMenu struct {
	Time     time.Time
	MenuData UpdateMenu
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// SetStatus is an event handler the updates the page status on the main page.
type LoginComplete struct {
	Time     time.Time
	Username string
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// SetStatus is an event handler the updates the page status on the main page.
type LogoutComplete struct {
	Time    time.Time
	Message string
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}
