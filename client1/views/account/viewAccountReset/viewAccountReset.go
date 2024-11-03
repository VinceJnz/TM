package viewAccountReset

import (
	"client1/v2/views/account/viewAccountModels"
	"log"
	"syscall/js"
	"time"
)

const debugTag = "viewAccountReset."

type UI struct {
	Notes           js.Value
	FromDate        js.Value
	ToDate          js.Value
	BookingStatusID js.Value
}

type ParentData struct {
	ID       int       `json:"id"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type Item struct {
	Record TableData
	//Add child structures as necessary
	//BookingPeople *bookingPeopleView.ItemEditor
}

type processStep int

const (
	getUsername processStep = iota
	getToken
	finished
)

// ItemView structure for  view
type ItemView struct {
	Item          *viewAccountModels.SrpItem
	Dispatcher    *Event.Dispatcher
	Message       viewAccountModels.Message
	PasswordChk   string
	Token         string // This is a temp value, used for testing account verification (instead using of email)
	ProcessStep   processStep
	FormValid     bool
	PasswordMatch bool
}

// New creates a new ItemView
func New(V *AppConfig) *ItemView {
	newView := new(ItemView)
	newView.Item = &viewAccountModels.SrpItem{}
	newView.Dispatcher = d
	newView.ProcessStep = getUsername
	newView.FormValid = false
	newView.PasswordMatch = false
	return newView
}

// onSubmit Send the username to the server
// The server sends a token via email if the username is valid
// Display the token input box once the usename has been sent and confirmed as valid
// onSubmit finish the process by sending the auth info to the server to update the auth info
func (p *ItemView) onSubmit(event *Event) {
	switch p.ProcessStep {
	case getUsername:
		//do something with the username
		p.Dispatcher.Dispatch(&storeUserAuth.AccountReset{Time: time.Now(), Item: *p.Item, CallbackSuccess: p.onSubmitOk, CallbackFail: p.onSubmitErr})
	case getToken:
		//do something with the token
		if p.Token != "" {
			p.Dispatcher.Dispatch(&storeUserAuth.AccountUpdate{Time: time.Now(), Token: p.Token, Item: *p.Item, CallbackSuccess: p.onSubmitOk, CallbackFail: p.onSubmitErr})
		}
	}
}

func (p *ItemView) onSubmitOk(svrMessage interface{}) {
	message := &viewAccountModels.Message{
		Id:     0,
		Text:   svrMessage.(string), //???? need to explain why ????? e.g. user name not found, or account locked?
		Status: viewAccountModels.MessageStatusInfo,
	}
	//Step forwards in the process....
	switch p.ProcessStep {
	case getUsername:
		p.ProcessStep = getToken
		p.FormValid = false
	case getToken:
		p.ProcessStep = finished
		p.Token = ""
		p.Item = &viewAccountModels.SrpItem{}
		p.FormValid = false
	case finished:
	}

	p.Dispatcher.Dispatch(&storeMessage.SetMessage{Item: message})
	//Message.Set(message) //need to record the message ID messageID := ...
	Rerender(p) //??????????????????????
}

func (p *ItemView) onSubmitErr(svrMessage interface{}) {
	//Item not valid
	message := &viewAccountModels.Message{
		Id:     0,
		Text:   "Username taken: " + svrMessage.(string), //p.Message.Text = err.Error() ???????????????
		Status: viewAccountModels.MessageStatusInfo,
	}
	//Step backwards in the process....
	switch p.ProcessStep {
	case getUsername:
	case getToken:
		p.ProcessStep = getUsername
		p.FormValid = false
	case finished:
	}

	p.Dispatcher.Dispatch(&storeMessage.SetMessage{Item: message})
	//Message.Set(message) //need to record the message ID messageID := ...
	Rerender(p) //??????????????????????
}

func (p *ItemView) onEditInput(event *Event) {
	//Get values from input, store in internal struct
	var label, value string
	label = event.Target.Get("id").String()
	value = event.Target.Get("value").String()

	switch label {
	case "Token":
		p.Token = value //Take some action when this is provided ?????????????????
	case "User Name":
		p.Item.UserName = value
	case "Password":
		p.Item.Password = value
	case "Re-enter Password":
		p.PasswordChk = value
	default:
		log.Println(debugTag+"ItemView.onEditInput()3 ", "p.Item =", p.Item, "label =", label, "value =", value)
		return //err????
	}

	p.FormValidation()
}

func (p *ItemView) FormValidation() {
	//Form validation
	p.FormValid = false
	switch p.ProcessStep {
	case getUsername:
		if p.Item.UserName != "" {
			p.FormValid = true
		} else {
			p.FormValid = false
		}
	case getToken:
		if p.Item.Password != "" {
			if p.Item.Password == p.PasswordChk {
				p.PasswordMatch = true
				if p.Item.UserName != "" {
					if p.Token != "" {
						p.FormValid = true
					}
				}
				//} else {
				//	p.FormValid = false
			} else {
				p.PasswordMatch = false
			}
		}
	}
	Rerender(p) //This can probably go as it would be triggered via a listner after the update to the store. ?????
}

// Render vecty render function
// Collect the user name, password and token from the user
func (p *ItemView) Render() {
	return Div(
		Markup(
			Class("form-group"),
		),
		Form(
			Markup(
				Submit(p.onSubmit).PreventDefault(), //Prevent default stops the form resetting the browser.!!!!!!!!
			),
			p.RenderRecordEdit(),
			Button(
				Markup(Disabled(!p.FormValid)),
				Text("Submit"),
			),
			Div( //message container is where server (or other) messages are displayed
				Markup(
					Class("vjMessage"),
				),
			),
		),
	)
}

// RenderRecordEdit vecty render function
func (p *ItemView) RenderRecordEdit() *HTML {
	return Div( //Record Edit
		Markup(
			Class("vjRecordEdit"),
			Class("vjEditing"),
		),
		viewHelper.RenderItemEdit(!(p.ProcessStep == getUsername), "User Name", p.Item.UserName, "Text", p.onEditInput),
		If(!(p.ProcessStep == getUsername),
			viewHelper.RenderItemEdit(!(p.ProcessStep == getToken), "Password", p.Item.Password, "Password", p.onEditInput),
			viewHelper.RenderItemEdit((p.Item.Password == "") || !(p.ProcessStep == getToken), "Re-enter Password", p.PasswordChk, "Password", p.onEditInput),
			viewHelper.RenderAlertMessage(!p.PasswordMatch, "passwords don't match"),
			viewHelper.RenderItemEdit(!(p.ProcessStep == getToken), "Token", p.Token, "Text", p.onEditInput),
		),
	)
}
