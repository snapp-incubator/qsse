package internal

import (
	"context"
	"log"
	"runtime"

	"github.com/mehditeymorian/koi"
)

const (
	DistributeEvent = "Distribute"
	AcceptClient    = "Accept"
)

type Worker struct {
	Pond *koi.Pond
}

type WorkerConfig struct {
	ClientAcceptorCount       int64
	ClientAcceptorQueueSize   int
	EventDistributorCount     int64
	EventDistributorQueueSize int
}

func NewWorker(cfg WorkerConfig) Worker {
	var worker Worker

	pond := koi.NewPond()
	worker.Pond = pond

	registerWorkers(pond, cfg)

	return worker
}

func registerWorkers(pond *koi.Pond, cfg WorkerConfig) {
	distributeWorker := koi.Worker{
		QueueSize:       cfg.EventDistributorQueueSize,
		ConcurrentCount: cfg.EventDistributorCount,
		Work:            distributeWork,
	}
	_ = pond.RegisterWorker(DistributeEvent, distributeWorker)

	acceptClientWorker := koi.Worker{
		QueueSize:       cfg.ClientAcceptorQueueSize,
		ConcurrentCount: cfg.ClientAcceptorCount,
		Work:            acceptClientWork,
	}
	_ = pond.RegisterWorker(AcceptClient, acceptClientWorker)
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

	eventSource.Metrics.DecEvent(topic)

	for _, subscriber := range eventSource.Subscribers {
		if subscriber.Corrupt.Load() {
			continue
		}

		if err := WriteData(event, subscriber.Stream); err != nil {
			log.Printf("err while sending event to client: %s", err.Error())
			subscriber.Corrupt.Store(true)
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

// acceptClientWork accepts clients and do the following steps.
// 1. Accept a receivedStream.
// 2. Read client authentication token and topics.
// 3. Authenticate the client.
// 3.1 If the authentication is successful, opens sendStream for each topic and add them to eventSources.
// 3.2 If the authentication is not successful, closes the connection.
func acceptClientWork(work any) any {
	runtime.LockOSThread()

	server, ok := work.(*Server)
	if !ok {
		return nil
	}

	for {
		background := context.Background()

		connection, err := server.Listener.Accept(background)
		if err != nil {
			log.Printf("failed to accept new client: %+v\n", err)

			continue
		}

		log.Println("found a new client")

		go server.handleClient(connection)
	}
}

func (w Worker) AddAcceptClientWork(server *Server, count int) {
	for i := 0; i < count; i++ {
		_, err := w.Pond.AddWork(AcceptClient, server)
		if err != nil {
			log.Printf("failed to add accept client work: %v\n", err)
		}
	}
}
