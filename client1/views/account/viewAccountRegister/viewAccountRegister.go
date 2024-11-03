package viewAccountRegister

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/account/viewAccountModels"
	"log"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "viewAccountRegister."

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

type TableData struct {
	Item *viewAccountModels.SrpItem
	//Dispatcher    *Event.Dispatcher
	Message       viewAccountModels.Message
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
	ItemState     viewAccountModels.ItemState
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
	editor.ItemState = viewAccountModels.ItemStateNone

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
	message := &viewAccountModels.Message{
		Id:     0,
		Text:   svrMessage.(string), //???? need to explain why ????? e.g. user name taken?
		Status: viewAccountModels.MessageStatusInfo,
	}
	p.Dispatcher.Dispatch(&storeMessage.SetMessage{Item: message})
	//Message.Set(message) //need to record the message ID messageID := ...
	//Rerender(p) //??????????????????????
}

func (p *ItemView) onSubmitErr(err interface{}) {
	log.Printf("%v %v %+v %v %+v", debugTag+"ItemView.onSubmitErr()1 ", "p.Item =", p.Item, "err =", err.(error))
	//Item not valid
	message := &viewAccountModels.Message{
		Id:     0,
		Text:   "Username taken: " + err.(error).Error(),
		Status: viewAccountModels.MessageStatusWarning,
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
