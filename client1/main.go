package main

import (
	"client1/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	view := mainView.New()
	view.Setup()

	<-c
}
