package main

import (
	"github.com/snapp-incubator/qsse"
	"log"
)

func main() {
	config := &qsse.ClientConfig{
		Token:           "secret",
		TLSConfig:       nil,
		ReconnectPolicy: nil,
	}
	topics := []string{"people.1.firstname", "people.2.firstname", "people.1.lastname", "people.2.lastname"}

	client, err := qsse.NewClient("localhost:4242", topics, config)
	if err != nil {
		panic(err)
	}

	client.SetEventHandler("people.1.firstname", func(bytes []byte) {
		log.Println("received on people.1.firstname")
	})

	client.SetEventHandler("people.*.firstname", func(bytes []byte) {
		log.Println("received on people.*.firstname")
	})

	client.SetEventHandler("people.*.lastname", func(bytes []byte) {
		log.Println("received on people.*.lastname")
	})

	select {}
}
