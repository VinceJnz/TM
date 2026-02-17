package mainView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
)

// ********************* This needs to be changed for each api **********************
const ApiURL = "/auth"

func (editor *View) MenuProcess() {
	// Next process step
	editor.getMenuUser()
}

// getMenuUser gets the menu user from the server (step 1) - The user needs to be logged on.
func (editor *View) getMenuUser() {
	//Get Menu User from server
	var menuUser appCore.User

	success := func(err error, data *httpProcessor.ReturnData) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()2 success: ", "err =", err, "MenuUser", editor.appCore.User) //Log the error in the browser
		}
		editor.appCore.SetUser(menuUser) // Save the menuUser to the appCore
		//editor.CurrentRecord.MenuUser = menuUser // Save the menuUser to the current record
		editor.elements.userDisplay.Set("innerHTML", editor.appCore.User.Name)
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateNone, debugTag)
		// Next process step
		editor.getMenuList()
		log.Printf("%v %v", debugTag+"getMenuUser()", "Menu user fetch complete")
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()4 fail: ", "err =", err, "MenuUser", editor.appCore.User) //Log the error in the browser
		//Don't display message to user
	}

	log.Printf("%v %v client = %+v httpClient = %+v", debugTag+"getMenuUser()", "Fetching menu user from server.", editor.client, editor.client.HTTPClient)
	editor.ItemState.UpdateStatus(viewHelpers.ItemStateFetching, debugTag)
	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuUser/", &menuUser, nil, success, fail)
	}()
}

// getMenuList gets the menu list from the server (step 2) - This is used to display or hide the menu buttons depending on the users level of access.
func (editor *View) getMenuList() {
	//Get Menu List from server
	var menuList MenuList

	success := func(err error, data *httpProcessor.ReturnData) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()2 success: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		}
		editor.CurrentRecord.MenuList = menuList // Save the salt to the current record
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateNone, debugTag)
		// Next process step
		editor.menuComplete()
		log.Printf("%v %v", debugTag+"getMenuList()", "Menu list fetch complete")
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()4 fail: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getMenuList()1 " + err.Error())
	}

	editor.ItemState.UpdateStatus(viewHelpers.ItemStateFetching, debugTag)
	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuList/", &menuList, nil, success, fail)
	}()
}

func (editor *View) menuComplete() {
	// Need to do something here to signify the menu data fetch being successful!!!!
	editor.onCompletionMsg(debugTag + "Menu fetch complete")
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateMenu", DebugTag: debugTag, Data: UpdateMenu{
		MenuUser: editor.appCore.User,
		MenuList: editor.CurrentRecord.MenuList,
	}})
}
