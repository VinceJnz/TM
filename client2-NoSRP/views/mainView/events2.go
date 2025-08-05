package mainView

import "fmt"

// Testing dispatcher ideas

type t1 struct {
	Name string
}

func (t t1) Handle(i interface{}) {
	d, ok := i.(t1)
	if ok {
		t.Name = d.Name
		fmt.Println("t1 - does something:", t.Name)
	} else {
		fmt.Println("t1 - Invalid data type")
	}
}

type t2 struct {
	Name string
	Age  int
}

func (t t2) Handle(i interface{}) {
	d, ok := i.(t2)
	if ok {
		t.Name = d.Name
		t.Age = d.Age
		fmt.Println("t2 - does something:", t.Name, t.Age)
	} else {
		fmt.Println("t2 - Invalid data type")
	}
}

type Listener interface {
	Handle(event interface{})
}

type Dispatcher struct {
	events map[Listener]Listener
}

func (d *Dispatcher) Add(e Listener) {
	d.events[e] = e
}

func (d *Dispatcher) Fire(e Listener, t interface{}) {
	d.events[e].Handle(t)
}

func Test() {
	D := Dispatcher{events: make(map[Listener]Listener)}

	var a1 t1
	var a2 t2

	D.Add(a1)
	D.Add(a2)

	D.Fire(a1, t1{Name: "test it"})
	D.Fire(a2, t2{Name: "test", Age: 2})
	D.Fire(a2, t1{Name: "test"})
}
