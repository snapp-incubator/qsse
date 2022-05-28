package main

import (
	"github.com/snapp-incubator/qsse"
)

func main() {
	config := &qsse.ClientConfig{
		Token:     "secret",
		TLSConfig: nil,
		ReconnectPolicy: qsse.ReconnectPolicy{
			Retry: false,
		},
	}
	topics := []string{"firstnames", "lastnames"}

	_, err := qsse.NewClient("localhost:4242", topics, config)
	if err != nil {
		panic(err)
	}

	select {}
}
