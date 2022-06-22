package internal

import (
	"log"
	"runtime"

	"github.com/Jeffail/tunny"
)

type Worker struct {
	SubscribePool *tunny.Pool
}

type SubscribeWork struct {
	Event       []byte
	EventSource *EventSource
}

func NewWorker() Worker {
	var worker Worker

	numCPU := runtime.NumCPU()

	worker.SubscribePool = tunny.NewFunc(numCPU, func(work any) any {
		data, ok := work.(*SubscribeWork)
		if !ok {
			log.Println("Worker: invalid work input")

			return nil
		}

		topic := data.EventSource.Topic
		eventData := data.Event
		eventSource := data.EventSource
		event := NewEvent(topic, eventData)

		eventSource.Metrics.IncDistributeEvent()
		eventSource.Metrics.DecEvent(topic)
		for _, subscriber := range eventSource.Subscribers {
			if subscriber.Corrupt.Load() {
				continue
			}

			if err := WriteData(event, subscriber.Stream); err != nil {
				log.Printf("err while sending event to client: %s", err.Error())
				subscriber.Corrupt.Store(true)
				eventSource.Metrics.DecSubscriber(topic)
			}
		}

		return nil
	})

	return worker
}

func NewSubscribeWork(event []byte, eventSource *EventSource) *SubscribeWork {
	return &SubscribeWork{Event: event, EventSource: eventSource}
}
