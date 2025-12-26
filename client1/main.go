package main

import (
	"client1/v2/app/appCore"
	"client1/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	appCore := appCore.New("https://localhost:8086/api/v1")
	defer appCore.Destroy() // ensure resources are cleaned up if main ever exits
	view := mainView.New(appCore)
	view.Setup()

	<-c
}
