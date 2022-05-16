package main

import (
	"flag"
	"fmt"

	"github.com/snapp-incubator/qsse"
)

func main() {
	client := 1
	flag.IntVar(&client, "client", client, "number of client")
	flag.Parse()

	topics := []string{"topic1", "topic2", "topic3"}

	for i := 0; i < client; i++ {
		go subscribe(i, topics)
	}

	select {}
}

func subscribe(clientID int, topics []string) {
	client, err := qsse.NewClient("localhost:8080", "client", topics)
	if err != nil {
		panic(err)
	}

	client.SetMessageHandler(func(topic string, message []byte) {
		fmt.Printf("client-%d %s: %s\n", clientID, topic, message)
	})

	select {}
}
