package mainView

import (
	"client1/v2/app/eventProcessor"
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

// getMenuUser gets the menu user from the server (step 1)
func (editor *View) getMenuUser() {
	//Get Menu User from server
	var menuUser MenuUser

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()2 success: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser
		}
		editor.CurrentRecord.MenuUser = menuUser // Save the menuUser to the current record
		editor.elements.userDisplay.Set("innerHTML", editor.CurrentRecord.MenuUser.Name)
		//log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()3 success: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser

		// Next process step
		editor.getMenuList()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()4 fail: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getMenuUser()1 " + err.Error())
	}

	go func() {
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuUser/", &menuUser, nil, success, fail)
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateNone)
	}()
}

// getMenuList gets the menu list from the server (step 2) - This is used to disply or hide the menu button depending on the users level of access to the url
func (editor *View) getMenuList() {
	//Get Menu List from server
	var menuList MenuList

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()2 success: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		}
		editor.CurrentRecord.MenuList = menuList // Save the salt to the current record
		//log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()3 success: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser

		// Next process step
		editor.menuComplete()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()4 fail: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getMenuList()1 " + err.Error())
	}

	go func() {
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuList/", &menuList, nil, success, fail)
		editor.ItemState.UpdateStatus(viewHelpers.ItemStateNone)
	}()
}

func (editor *View) menuComplete() {
	// Need to do something here to signify the menu data fetch being successful!!!!
	//log.Printf("%v %v %+v %v %+v", debugTag+"loginComplete()1 ", "MenuUser =", editor.CurrentRecord.MenuUser, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser

	editor.onCompletionMsg(debugTag + "menuComplete()2 successfully completed menu fetch:")
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateMenu", DebugTag: debugTag, Data: UpdateMenu{
		MenuUser: editor.CurrentRecord.MenuUser,
		MenuList: editor.CurrentRecord.MenuList,
	}})
}
