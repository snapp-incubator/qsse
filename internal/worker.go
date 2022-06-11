package internal

import (
	"go.uber.org/zap"
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

func NewWorker(l *zap.Logger) Worker {
	var worker Worker

	numCPU := runtime.NumCPU()

	worker.SubscribePool = tunny.NewFunc(numCPU, func(work any) any {
		data, ok := work.(*SubscribeWork)
		if !ok {
			l.Warn("Worker: invalid work input")

			return nil
		}

		topic := data.EventSource.Topic
		event := data.Event
		eventSource := data.EventSource

		for i, subscriber := range eventSource.Subscribers {
			l.Info("Sending event to subscriber for topic", zap.String("topic", topic))
			event := NewEvent(topic, event)
			err := WriteData(event, subscriber)
			if err != nil {
				log.Printf("err while sending event to client: %s", err.Error())
				eventSource.Subscribers = append(eventSource.Subscribers[:i], eventSource.Subscribers[i+1:]...)
			}
		}

		return nil
	})

	return worker
}

func NewSubscribeWork(event []byte, eventSource *EventSource) *SubscribeWork {
	return &SubscribeWork{Event: event, EventSource: eventSource}
}
