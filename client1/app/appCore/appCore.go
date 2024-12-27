package appcore

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"net/http"
	"syscall/js"
)

const debugTag = "appCore."

// AppCore contains the elements used by all the views

const ApiURL = "/auth"

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

type MenuData struct {
	MenuUser MenuUser
	MenuList MenuList
}

type AppCore struct {
	HttpClient *httpProcessor.Client
	Events     *eventProcessor.EventProcessor
	Document   js.Value
	MenuData   MenuData
}

func New(httpClient *httpProcessor.Client) *AppCore {
	ac := &AppCore{}
	ac.HttpClient = httpProcessor.New("https://localhost:8086/api/v1")
	ac.Events = eventProcessor.New()
	ac.Document = js.Global().Get("document")

	window := js.Global().Get("window")
	window.Call("addEventListener", "onbeforeunload", js.FuncOf(ac.BeforeUnload))

	log.Printf("%v %v", debugTag+"New()", "AppCore created")
	return ac
}

// ********************* This needs to be changed for each api **********************

func (ac *AppCore) BeforeUnload(this js.Value, args []js.Value) interface{} {
	log.Printf(debugTag + "New()1 onbeforeunload")
	return nil
}

func (ac *AppCore) MenuProcess() {
	// Next process step
	ac.getMenuUser()
}

// getMenuUser gets the menu user from the server (step 1)
func (ac *AppCore) getMenuUser() {
	//Get Menu User from server
	var menuUser MenuUser

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()2 success: ", "err =", err, "MenuUser", ac.MenuData.MenuUser) //Log the error in the browser
		}
		ac.MenuData.MenuUser = menuUser // Save the menuUser to the current record
		//log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()3 success: ", "err =", err, "MenuUser", editor.CurrentRecord.MenuUser) //Log the error in the browser

		// Next process step
		ac.getMenuList()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuUser()4 fail: ", "err =", err, "MenuUser", ac.MenuData.MenuUser) //Log the error in the browser
		//Don't display message to user
	}

	go func() {
		ac.HttpClient.NewRequest(http.MethodGet, ApiURL+"/menuUser/", &menuUser, nil, success, fail)
	}()
}

// getMenuList gets the menu list from the server (step 2) - This is used to display or hide the menu buttons depending on the users level of access.
func (ac *AppCore) getMenuList() {
	//Get Menu List from server
	var menuList MenuList

	success := func(err error) {
		//Call the next step in the Auth process
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()2 success: ", "err =", err, "MenuList =", ac.MenuData.MenuList) //Log the error in the browser
		}
		ac.MenuData.MenuList = menuList // Save the salt to the current record
		//log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()3 success: ", "err =", err, "MenuList =", editor.CurrentRecord.MenuList) //Log the error in the browser

		// Next process step
		//ac.menuComplete()
	}

	fail := func(err error) {
		log.Printf("%v %v %v %v %+v", debugTag+"LogonForm.getMenuList()4 fail: ", "err =", err, "MenuList =", ac.MenuData.MenuList) //Log the error in the browser
		//Display message  to user ??????????????
	}

	go func() {
		ac.HttpClient.NewRequest(http.MethodGet, ApiURL+"/menuList/", &menuList, nil, success, fail)
	}()
}
