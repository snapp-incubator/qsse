package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/snapp-incubator/qsse"
)

func main() {
	generatorsCount := 1
	publishRate := 1000

	flag.IntVar(&generatorsCount, "generators", 1, "number of generators")
	flag.IntVar(&publishRate, "rate", 1000, "publish rate in milliseconds")
	flag.Parse()

	rate := time.Duration(publishRate) * time.Millisecond

	topics := []string{"topic1", "topic2", "topic3"}

	server, err := qsse.NewServer("localhost:8080", topics, nil)
	if err != nil {
		panic(err)
	}

	for range generatorsCount {
		go generate(topics[rand.Intn(len(topics))], server, rate)
	}

	select {}
}
