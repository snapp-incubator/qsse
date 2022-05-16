package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/snapp-incubator/qsse"
)

var firstNames = []string{"Harry", "Ross", "Bruce", "Cook", "Carolyn", "Morgan",
	"Albert", "Walker", "Randy", "Reed", "Larry", "Barnes", "Lois", "Wilson",
	"Jesse", "Campbell", "Ernest", "Rogers", "Theresa", "Patterson", "Henry",
	"Simmons", "Michelle", "Perry", "Frank", "Butler", "Shirley"}

var lastNames = []string{"Ruth", "Jackson", "Debra", "Allen", "Gerald", "Harris",
	"Raymond", "Carter", "Jacqueline", "Torres", "Joseph", "Nelson", "Carlos",
	"Sanchez", "Ralph", "Clark", "Jean", "Alexander", "Stephen", "Roberts",
	"Eric", "Long", "Amanda", "Scott", "Teresa", "Diaz", "Wanda", "Thomas"}

func main() {
	authenticateFunc := func(token string) bool {
		log.Printf("Authenticating token: %s", token)
		return token == "secret"
	}

	topics := []string{"firstnames", "lastnames"}

	server, err := qsse.NewServer("localhost:4242", qsse.GetDefaultTLSConfig(), topics)
	if err != nil {
		panic(err)
	}
	server.SetAuthentication(authenticateFunc)

	go func() {
		for {
			if rand.NormFloat64() > 0.5 {
				server.Publish("firstnames", []byte(firstNames[rand.Intn(len(firstNames))]))
			} else {
				server.Publish("lastnames", []byte(lastNames[rand.Intn(len(lastNames))]))
			}
			<-time.After(2 * time.Second)
		}
	}()

	select {}
}
