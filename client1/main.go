package main

import (
	"client1/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	view := mainView.New(mainView.AppConfig{BaseURL: "http://localhost:8085/api/v1"})
	view.Setup()

	<-c
}
