package main

import (
	"client1/v2/app/httpProcessor"
	"client1/v2/views/mainView"
	"log"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	//client := httpProcessor.New("http://localhost:9080/api/v1")
	client := httpProcessor.New("https://localhost:9443/api/v1")
	view := mainView.New(client)
	view.Setup()

	<-c
}
