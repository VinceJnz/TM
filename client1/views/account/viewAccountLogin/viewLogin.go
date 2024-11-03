package viewAccountLogin

import (
	//"log"

	"log"
	"time"

	"github.com/1Password/srp"
)

const debugTag = "viewAccountLogin."

type LogonForm struct {
	callbackLoginOk      func(error, mdlUser.SrpItem)
	callbackLoginErr     func(error)
	callbackNewAccount   func(error)
	callbackAccountReset func(error)
	Dispatcher           *Event.Dispatcher
	Group                int
	Item                 mdlUser.SrpItem //?????????????????

	LoggedIn  bool
	srpClient *srp.SRP
	FormValid bool
}

func New(d *Event.Dispatcher) *LogonForm {
	return &LogonForm{
		//User:         &s.User.Item,
		//AppState:     &s.AppState.Structure,
		Dispatcher: d,
		Group:      srp.RFC5054Group3072,
		//Item:         mdlUser.SrpItem{}, //????????????????????
	}
}

func (s *LogonForm) Config(callbackLoginOk func(error, mdlUser.SrpItem), callbackLoginErr, newAccountMenu func(error), accountReset func(error)) {
	s.callbackLoginOk = callbackLoginOk
	s.callbackLoginErr = callbackLoginErr
	s.callbackNewAccount = newAccountMenu
	s.callbackAccountReset = accountReset
	s.Item = mdlUser.SrpItem{} //This resets the values in the login form ???????????????????
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

func (s *LogonForm) Render() ComponentOrHTML {
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
