package loginView

import (
	"client1/v2/app/eventProcessor"
	"log"
	"net/http"
)

func (editor *ItemEditor) MenuProcess() {
	// Next process step
	editor.getMenuUser()
}

// getMenuUser gets the menu user from the server (step 1)
func (editor *ItemEditor) getMenuUser() {
	//Get Menu User from server
	var menuUser MenuUser

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()3 success: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser
		}
		editor.CurrentRecord.MenuUser = menuUser // Save the menuUser to the current record

		// Next process step
		editor.getMenuList()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()4 fail: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getMenuUser()1 " + err.Error())
	}

	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuUser/", &menuUser, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()
}

// getMenuList gets the menu list from the server (step 2)
func (editor *ItemEditor) getMenuList() {
	//Get Menu List from server
	var menuList MenuList

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()3 success: ", "err =", err, "MenuUser =", editor.CurrentRecord.MenuUser) //Log the error in the browser
		}
		editor.CurrentRecord.MenuList = menuList // Save the salt to the current record

		// Next process step
		editor.menuComplete()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()4 fail: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser
		//Display message  to user ??????????????
		editor.onCompletionMsg(debugTag + "getMenuList()1 " + err.Error())
	}

	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/menuList/", &menuList, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()
}

func (editor *ItemEditor) menuComplete() {
	// Need to do something here to signify the menu data fetch being successful!!!!
	log.Printf("%v %v %+v %v %+v", debugTag+"loginComplete()1 ", "MenuUser =", editor.CurrentRecord.MenuUser, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser

	editor.onCompletionMsg(debugTag + "menuComplete()2 successfully completed menu fetch:")
	editor.events.ProcessEvent(eventProcessor.Event{Type: "menuDataFetch", Data: UpdateMenu{
		MenuUser: editor.CurrentRecord.MenuUser,
		MenuList: editor.CurrentRecord.MenuList,
	}})
}
