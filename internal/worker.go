package internal

import (
	"log"

	"github.com/mehditeymorian/koi"
)

const (
	DistributeEvent = "Distribute"
)

type Worker struct {
	Pond *koi.Pond
}

func NewWorker() Worker {
	var worker Worker

	pond := koi.NewPond()
	worker.Pond = pond

	registerWorkers(pond)

	return worker
}

func registerWorkers(pond *koi.Pond) {
	distributeWorker := koi.Worker{
		QueueSize:       100,
		ConcurrentCount: 10,
		Work:            distributeWork,
	}
	_ = pond.RegisterWorker(DistributeEvent, distributeWorker)
}

// ---------------------------------------------------------------

type DistributeWork struct {
	Event       []byte
	EventSource *EventSource
}

func NewSubscribeWork(event []byte, eventSource *EventSource) *DistributeWork {
	return &DistributeWork{Event: event, EventSource: eventSource}
}

func distributeWork(work any) any {
	data, ok := work.(*DistributeWork)
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
}

func (w *Worker) AddDistributeWork(work *DistributeWork) {
	_, err := w.Pond.AddWork(DistributeEvent, work)
	if err != nil {
		log.Printf("failed to add distribute work: %v\n", err)
	}
}

// ---------------------------------------------------------------
