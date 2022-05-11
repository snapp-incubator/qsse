package main

import (
	"github.com/mehditeymorian/qsse"
)

func main() {
	_, err := qsse.NewClient("localhost:4242", "secret", []string{"test", "test2"})
	if err != nil {
		panic(err)
	}

	select {}
}
