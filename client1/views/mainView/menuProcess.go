package mainView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
)

const ApiURL = "/auth"

func (editor *View) setMenuFetchState(state viewHelpers.ItemState) {
	editor.ItemState.UpdateStatus(state, debugTag)
}

func (editor *View) requestMenuData(path string, target any, success func(error), fail func(error)) {
	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURL+path, target, nil, success, fail)
	}()
}

func (editor *View) MenuProcess() {
	editor.getMenuUser()
}

// getMenuUser gets the menu user from the server (step 1) - The user needs to be logged on.
func (editor *View) getMenuUser() {
	//Get Menu User from server
	var menuUser appCore.User

	success := func(err error) {
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"getMenuUser()2 success: ", "err =", err, "MenuUser", editor.appCore.User) //Log the error in the browser
		}
		editor.appCore.SetUser(menuUser) // Save the menuUser to the appCore
		editor.authResolved = true
		editor.elements.userDisplay.Set("innerHTML", editor.appCore.User.Name)
		editor.setMenuFetchState(viewHelpers.ItemStateNone)
		if menuUser.UserID > 0 {
			editor.getMenuList()
		} else {
			editor.resetMenu()
		}
		editor.applyRouteFromLocation()
		log.Printf("%v %v", debugTag+"getMenuUser()", "Menu user fetch complete")
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"getMenuUser()4 fail: ", "err =", err, "MenuUser", editor.appCore.User) //Log the error in the browser
		editor.authResolved = true
		editor.appCore.SetUser(appCore.User{})
		editor.resetMenu()
		editor.setMenuFetchState(viewHelpers.ItemStateNone)
		editor.applyRouteFromLocation()
		//Don't display message to user
	}

	log.Printf("%v %v client = %+v httpClient = %+v", debugTag+"getMenuUser()", "Fetching menu user from server.", editor.client, editor.client.HTTPClient)
	editor.setMenuFetchState(viewHelpers.ItemStateFetching)
	editor.requestMenuData("/menuUser/", &menuUser, success, fail)
}

// getMenuList gets the menu list from the server (step 2) - This is used to display or hide the menu buttons depending on the users level of access.
func (editor *View) getMenuList() {
	//Get Menu List from server
	var menuList MenuList

	success := func(err error) {
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"getMenuList()2 success: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		}
		editor.CurrentRecord.MenuList = menuList // Save the salt to the current record
		capabilities := make([]appCore.Capability, 0, len(menuList))
		for _, item := range menuList {
			capabilities = append(capabilities, appCore.Capability{
				Resource:    item.Resource,
				AccessLevel: item.AccessLevel,
				AccessScope: item.AccessScope,
			})
		}
		editor.appCore.SetCapabilities(capabilities)
		editor.setMenuFetchState(viewHelpers.ItemStateNone)
		editor.menuComplete()
		log.Printf("%v %v", debugTag+"getMenuList()", "Menu list fetch complete")
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"getMenuList()4 fail: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		editor.onCompletionMsg(debugTag + "getMenuList()1 " + err.Error())
	}

	editor.setMenuFetchState(viewHelpers.ItemStateFetching)
	editor.requestMenuData("/menuList/", &menuList, success, fail)
}

func (editor *View) menuComplete() {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeUpdateMenu, DebugTag: debugTag, Data: editor.CurrentRecord.MenuList})
}
