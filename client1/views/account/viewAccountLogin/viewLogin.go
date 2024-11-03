package viewAccountLogin

import (
	//"log"

	"client1/v2/app/eventProcessor"
	"client1/v2/views/account/viewAccountModels"
	"log"
	"syscall/js"
	"time"

	"github.com/1Password/srp"
)

const debugTag = "viewAccountLogin."

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

// ********************* This needs to be changed for each api **********************
const apiURL = "/bookings"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID              int       `json:"id"`
	OwnerID         int       `json:"owner_id"`
	TripID          int       `json:"trip_id"`
	PersonID        int       `json:"person_id"`
	Notes           string    `json:"notes"`
	FromDate        time.Time `json:"from_date"`
	ToDate          time.Time `json:"to_date"`
	Participants    int       `json:"participants"` // Report generated field
	GroupBookingID  int       `json:"group_booking_id"`
	GroupBooking    string    `json:"group_booking"` // Report generated field
	BookingStatusID int       `json:"booking_status_id"`
	BookingStatus   string    `json:"booking_status"` // Report generated field
	BookingDate     time.Time `json:"booking_date"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
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

type ItemEditor struct {
	document      js.Value
	events        *eventProcessor.EventProcessor
	baseURL       string
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	StateDiv      js.Value

	//callbackLoginOk      func(error, mdlUser.SrpItem)
	//callbackLoginErr     func(error)
	//callbackNewAccount   func(error)
	//callbackAccountReset func(error)
	//Dispatcher           *Event.Dispatcher
	Group int
	//Item                 mdlUser.SrpItem //?????????????????

	LoggedIn  bool
	srpClient *srp.SRP
	FormValid bool
}

//func New(d *Event.Dispatcher) *LogonForm {
//	return &LogonForm{
//		//User:         &s.User.Item,
//		//AppState:     &s.AppState.Structure,
//		Dispatcher: d,
//		Group:      srp.RFC5054Group3072,
//		//Item:         mdlUser.SrpItem{}, //????????????????????
//	}
//}

func (s *LogonForm) Config(callbackLoginOk func(error, viewAccountModels.SrpItem), callbackLoginErr, newAccountMenu func(error), accountReset func(error)) {
	s.callbackLoginOk = callbackLoginOk
	s.callbackLoginErr = callbackLoginErr
	s.callbackNewAccount = newAccountMenu
	s.callbackAccountReset = accountReset
	s.Item = viewAccountModels.SrpItem{} //This resets the values in the login form ???????????????????
}

func (s *LogonForm) FormValidiation() {
	//Form validation - Login
	if s.Item.Password == "" || s.Item.UserName == "" {
		s.FormValid = false
	} else {
		s.FormValid = true
	}
	//log.Printf("%v %v %+v", debugTag+"LogonForm.FormValidiation()1", "s =", s)
	Rerender(s)
}

func (s *LogonForm) onSubmit(event *Event) {
	//log.Printf("%v %v %+v", debugTag+"LogonForm.onSubmit()1", "s =", s)
	s.authProcess() //Run the auth process
}

func (s *LogonForm) loginOk(err error) {
	if err != nil {
		log.Printf("%v %v %+v", debugTag+"LogonForm.loginOk()1", "s =", s)
	}
	s.Dispatcher.Dispatch(&storeUserAuth.SetUserName{Time: time.Now(), UserName: s.Item.UserName}) // this doesn't do anything useful ????????????????
	s.callbackLoginOk(err, s.Item)
}

func (s *LogonForm) loginErr(err error) {
	s.callbackLoginErr(err)
}

func (s *LogonForm) usernameOnChange(event *Event) {
	s.Item.UserName = event.Target.Get("value").String()
	s.FormValidiation()
}

func (s *LogonForm) passwordOnChange(event *Event) {
	s.Item.Password = event.Target.Get("value").String()
	s.FormValidiation()
}

func (s *LogonForm) newAccount(event *Event) {
	//Need to load the new account form
	s.callbackNewAccount(nil)
}

func (s *LogonForm) accountReset(event *Event) {
	//Need to load the password reset form
	s.callbackAccountReset(nil)
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, baseURL string, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.baseURL = baseURL
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.RecordState = RecordStateReloadRequired

	//editor.BookingStatus = bookingStatusView.New(editor.document, eventProcessor, baseURL)
	//editor.BookingStatus.FetchItems()

	//editor.PeopleEditor = bookingPeopleView.New(editor.document, editor.events, baseURL)

	return editor
}

func (s *LogonForm) Render() {
	return Div(
		Markup(
			Class("vjLoginEdit"),
		),
		Form(
			Markup(
				Submit(s.onSubmit).PreventDefault(),
			),
			RenderItemEdit(false, "Username", s.Item.UserName, "text", s.usernameOnChange),
			RenderItemEdit(false, "Password", s.Item.Password, "Password", s.passwordOnChange),
			viewHelper.RenderButton(!s.FormValid, "Login", "login to your account", TypeSubmit),
		),
		Paragraph(
			//Text(),
			viewHelper.RenderTextClick(false, "Reset password?", "Forgot your password password or password needs to be reset", s.accountReset),
		),
		Paragraph(
			Text("Not a customer? "),
			viewHelper.RenderTextClick(false, "Register now", "Register a new account", s.newAccount),
		),
	)
}

// RenderLoginItemEdit ??
func RenderItemEdit(disabled bool, label, value string, inputType string, onEdit func(*Event)) *HTML {
	return Div(
		Markup(
			Class("vjLoginItemEdit"),
		),
		Label(
			Markup(
				For(label), //?????????????????????
			),
			Text(label),
		),
		Input(
			Markup(
				Class("form-control"),
				ID(label),
				Value(value),
				Type(InputType(inputType)),
				Disabled(disabled),
				Input(onEdit),
			),
		),
	)
}
