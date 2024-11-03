package viewAccountLogout

import (
	"log"
	"time"
)

const debugTag = "viewAccountLogout."

type Page struct {
	loggedIn   bool
	Dispatcher *Event.Dispatcher
}

func NewXX(d *Event.Dispatcher, callback func()) *Page {
	callbackSuccess := func(err error) {
		log.Println(debugTag+"New()1 callbackSuccess", "err =", err)
		if err != nil {
			//log.Println(debugTag+"callbackSuccess()", "err =", err)
			return
		}
		if callback != nil {
			callback()
		}
	}

	newPage := &Page{
		Dispatcher: d,
	}
	newPage.Dispatcher.Dispatch(&storeUserAuth.Logout{Time: time.Now(), CallbackSuccess: callbackSuccess})
	return newPage
}

func (s *Page) Render() {
	return Div(
		Markup(
			ClassMap{
				"vjEditing": s.loggedIn,
			},
		),
		Text("User logged out"),
	)
}
