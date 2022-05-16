package main

import (
	"math/rand"
	"time"

	"github.com/snapp-incubator/qsse"
)

func RandomText(length int) []byte {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = characters[rand.Intn(len(characters))]
	}

	return bytes
}

func generate(topic string, server *qsse.Server, rate time.Duration) {
	for {
		server.PublishEvent(topic, RandomText(10))

		time.Sleep(rate)
	}
}
