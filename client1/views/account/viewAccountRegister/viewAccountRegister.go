package viewAccountRegister

import (
	"client1/v2/app/eventProcessor"
	"log"
	"math/big"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "viewAccountRegister."

type ItemState int

const (
	ItemStateNone     ItemState = iota
	ItemStateFetching           //ItemState = 1
	ItemStateEditing            //ItemState = 2
	ItemStateAdding             //ItemState = 3
	ItemStateSaving             //ItemState = 4
	ItemStateDeleting           //ItemState = 5
	ItemStateSubmitted
)

type ViewState int

const (
	ViewStateNone ViewState = iota
	ViewStateBlock
)

type RecordState int

const (
	RecordStateReloadRequired RecordState = iota
	RecordStateCurrent
)

type MessageStatus int

const (
	MessageStatusEmpty MessageStatus = iota
	MessageStatusInfo
	MessageStatusWarning
	MessageStatusError
)

type Message struct {
	Id     int64
	Text   string
	Status MessageStatus
}

// ServerVerify contains the verify info sent from the server
type ServerVerify struct {
	B     *big.Int `json:"B"`
	Proof []byte   `json:"Proof"`
	Token string   `json:"Token"`
}

// ClientVerify contains the clinet SRP verify info and is sent to the server
type ClientVerify struct {
	UserName string `json:"UserName"`
	Proof    []byte `json:"Proof"`
	Token    string `json:"Token"`
}

// SrpItem contains the user SRP info
type SrpItem struct {
	Item
	Salt     []byte   `json:"Salt"`     //Not user editable
	Verifier *big.Int `json:"Verifier"` //Not user editable
	Password string   `json:"Password"` //srp means this is not longer needed.
	Server   ServerVerify
	Client   ClientVerify
}

type TableData struct {
	Item *SrpItem
	//Dispatcher    *Event.Dispatcher
	Message       Message
	PasswordChk   string
	FormValid     bool
	PasswordMatch bool
	//Token      string          // This is a temp value, used for testing account verification (instead using of email)
}

type Item struct {
	Record TableData
	//Add child structures as necessary
}

type ItemEditor struct {
	document      js.Value
	events        *eventProcessor.EventProcessor
	baseURL       string
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	//config     AppConfig
}

// func New(d *Event.Dispatcher) *ItemView {
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, baseURL string, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.baseURL = baseURL
	editor.ItemState = ItemStateNone

	return editor
}

func (p *ItemView) onSubmit() {
	//Check if item is valid
	// if yes
	//   return to the logon page.
	//   Create new item and flag it as new, inform the user via email
	//   Wait for the user to validate the email before activating the account
	// if no
	//   return to new account page
	//   disply error message

	if p.Item.Password != p.PasswordChk {
		log.Printf("%v %v %+v %v %v", debugTag+"ItemView.onSubmit()1 ", "p.Item =", p.Item, "p.PasswordChk =", p.PasswordChk)
		return
	}
	p.Dispatcher.Dispatch(&storeUserAuth.NewAccountCreate{Time: time.Now(), Item: *p.Item, CallbackSuccess: p.onSubmitOk, CallbackFail: p.onSubmitErr})
}

func (p *ItemView) onSubmitOk(svrMessage interface{}) {
	message := &mdlMessage.Item{
		Id:     0,
		Text:   svrMessage.(string), //???? need to explain why ????? e.g. user name taken?
		Status: mdlMessage.StatusInfo,
	}
	p.Dispatcher.Dispatch(&storeMessage.SetMessage{Item: message})
	//Message.Set(message) //need to record the message ID messageID := ...
	//Rerender(p) //??????????????????????
}

func (p *ItemView) onSubmitErr(err interface{}) {
	log.Printf("%v %v %+v %v %+v", debugTag+"ItemView.onSubmitErr()1 ", "p.Item =", p.Item, "err =", err.(error))
	//Item not valid
	message := &mdlMessage.Item{
		Id:     0,
		Text:   "Username taken: " + err.(error).Error(),
		Status: mdlMessage.StatusWarning,
	}
	p.Dispatcher.Dispatch(&storeMessage.SetMessage{Item: message})
	//Message.Set(message) //need to record the message ID messageID := ...
	//Rerender(p) //??????????????????????
}

func (p *ItemView) onEditInput() {
	//Get values from input, store in internal struct
	var label, value string
	label = event.Target.Get("id").String()
	value = event.Target.Get("value").String()

	switch label {
	case "ID":
		p.Item.ID, _ = strconv.ParseInt(value, 10, 64)
	//case "UserStatusID":
	//	p.Item.StatusID, _ = strconv.ParseInt(value, 10, 64)
	case "Display Name":
		p.Item.DisplayName = value
	case "User Name":
		p.Item.UserName = value
	case "Email":
		p.Item.Email = value
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
	if p.Item.Password != "" {
		if p.Item.Password == p.PasswordChk {
			p.PasswordMatch = true
			if p.Item.Email != "" {
				if p.Item.UserName != "" {
					if p.Item.DisplayName != "" {
						p.FormValid = true
					}
				}
			}
			//} else {
			//	p.FormValid = false
		} else {
			p.PasswordMatch = false
		}
	}

	//Rerender(p) //This can probably go as it would be triggered via a listner after the update to the store. ?????
}

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
				Text("Create"),
			),
			Div( //message container
				Markup(
					Class("vjMessage"),
				),
			),
		),
	)
}

func (p *ItemView) RenderRecordEdit() {
	return Div( //Record Edit
		Markup(
			Class("vjRecordEdit"),
			Class("vjEditing"),
		),
		viewHelper.RenderItemEdit(false, "User Name", p.Item.UserName, "Text", p.onEditInput),
		viewHelper.RenderItemEdit(false, "Display Name", p.Item.DisplayName, "Text", p.onEditInput),
		viewHelper.RenderItemEdit(false, "Email", p.Item.Email, "email", p.onEditInput),
		viewHelper.RenderItemEdit(false, "Password", p.Item.Password, "Password", p.onEditInput),
		viewHelper.RenderItemEdit((p.Item.Password == ""), "Re-enter Password", p.PasswordChk, "Password", p.onEditInput),
		viewHelper.RenderAlertMessage(!p.PasswordMatch, "passwords don't match"),
	)
}
