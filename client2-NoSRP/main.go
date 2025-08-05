package main

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	appCore := appCore.New(httpProcessor.New("https://localhost:8086/api/v1"))
	view := mainView.New(appCore)
	view.Setup()

	<-c
}
