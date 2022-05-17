package main

import (
	"github.com/snapp-incubator/qsse/pkg"
	"math/rand"
	"time"
)

func RandomText(length int) []byte {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = characters[rand.Intn(len(characters))]
	}

	return bytes
}

func generate(topic string, server pkg.Server, rate time.Duration) {
	for {
		server.Publish(topic, RandomText(10))

		time.Sleep(rate)
	}
}
