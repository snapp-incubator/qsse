package main

import (
	"flag"
	"github.com/snapp-incubator/qsse"
	"math/rand"
	"time"
)

func main() {
	generatorsCount := 1
	publishRate := 1000

	flag.IntVar(&generatorsCount, "generators", 1, "number of generators")
	flag.IntVar(&publishRate, "rate", 1000, "publish rate in milliseconds")
	flag.Parse()

	rate := time.Duration(publishRate) * time.Millisecond

	topics := []string{"topic1", "topic2", "topic3"}

	server, err := qsse.NewServer("localhost:8080", qsse.GetDefaultTLSConfig(), topics)
	if err != nil {
		panic(err)
	}

	for i := 0; i < generatorsCount; i++ {
		go generate(topics[rand.Intn(len(topics))], server, rate)
	}

	select {}
}
