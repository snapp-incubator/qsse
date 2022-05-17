package main

import (
	"github.com/snapp-incubator/qsse/pkg"
)

func main() {
	config := &pkg.ClientConfig{
		Token:     "secret",
		TLSConfig: nil,
	}
	topics := []string{"firstnames", "lastnames"}

	_, err := pkg.NewClient("localhost:4242", topics, config)
	if err != nil {
		panic(err)
	}

	select {}
}
