package main

import (
	"github.com/mehditeymorian/qsse"
)

func main() {
	_, err := qsse.NewClient("localhost:4242", "secret", []string{"firstnames", "lastnames"})
	if err != nil {
		panic(err)
	}

	select {}
}
