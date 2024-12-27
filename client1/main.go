package main

import (
	appcore "client1/v2/app/appCore"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	appcore := appcore.New(httpProcessor.New("https://localhost:8086/api/v1"))
	view := mainView.New(appcore)
	view.Setup()

	<-c
}
