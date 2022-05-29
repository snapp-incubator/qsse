package main

import (
	"fmt"
	"github.com/snapp-incubator/qsse"
	"log"
	"math/rand"
	"time"
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

	topics := []string{"people.1.firstname", "people.2.firstname", "people.1.lastname", "people.2.lastname"}

	server, err := qsse.NewServer("localhost:4242", qsse.GetDefaultTLSConfig(), topics)
	if err != nil {
		panic(err)
	}
	server.SetAuthenticatorFunc(authenticateFunc)

	go func() {
		for {
			personID := rand.Intn(2) + 1
			if rand.NormFloat64() > 0.5 {
				server.Publish(fmt.Sprintf("people.%d.firstname", personID), RandomItem(firstNames))
			} else {
				server.Publish(fmt.Sprintf("people.%d.lastname", personID), RandomItem(lastNames))
			}
			<-time.After(2 * time.Second)
		}
	}()

	select {}
}

func RandomItem(items []string) []byte {
	return []byte(items[rand.Intn(len(items))])
}
