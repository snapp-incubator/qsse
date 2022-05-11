package main

import (
	"github.com/mehditeymorian/qsse"
	"log"
	"math/rand"
	"time"
)

func main() {
	authenticateFunc := func(token string) bool {
		log.Printf("Authenticating token: %s", token)
		return token == "secret"
	}

	server, err := qsse.NewServer("localhost:4242", qsse.GetDefaultTLSConfig(), authenticateFunc)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if rand.NormFloat64() > 0.5 {
				server.PublishEvent("test", []byte("higher chance to get a message"))
			} else {
				server.PublishEvent("23434", []byte("test2"))
			}
			<-time.After(5 * time.Second)
		}
	}()

	select {}
}
