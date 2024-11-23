package internal

import (
	"context"
	"runtime"

	"github.com/mehditeymorian/koi"
	"go.uber.org/zap"
)

const (
	DistributeEvent = "Distribute"
	AcceptClient    = "Accept"
)

type Worker struct {
	Pond   *koi.Pond
	Logger *zap.Logger
}

type WorkerConfig struct {
	ClientAcceptorCount       int64
	ClientAcceptorQueueSize   int
	EventDistributorCount     int64
	EventDistributorQueueSize int
}

func NewWorker(cfg WorkerConfig, l *zap.Logger) Worker {
	var worker Worker

	worker.Logger = l

	pond := koi.NewPond()
	worker.Pond = pond

	worker.registerWorkers(cfg)

	return worker
}

func (w *Worker) registerWorkers(cfg WorkerConfig) {
	distributeWorker := koi.Worker{
		QueueSize:       cfg.EventDistributorQueueSize,
		ConcurrentCount: cfg.EventDistributorCount,
		Work:            w.distributeWork,
	}
	_ = w.Pond.RegisterWorker(DistributeEvent, distributeWorker)

	acceptClientWorker := koi.Worker{
		QueueSize:       cfg.ClientAcceptorQueueSize,
		ConcurrentCount: cfg.ClientAcceptorCount,
		Work:            w.acceptClientWork,
	}
	_ = w.Pond.RegisterWorker(AcceptClient, acceptClientWorker)
}

// ---------------------------------------------------------------

type DistributeWork struct {
	Event       []byte
	EventSource *EventSource
}

func NewDistributeWork(event []byte, eventSource *EventSource) *DistributeWork {
	return &DistributeWork{Event: event, EventSource: eventSource}
}

func (w *Worker) distributeWork(work any) any {
	data, ok := work.(*DistributeWork)
	if !ok {
		w.Logger.Warn("Worker: invalid work input")

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
			w.Logger.Warn("err while sending event to client", zap.Error(err))
			subscriber.Corrupt.Store(true)
		}
	}

	return nil
}

func (w *Worker) AddDistributeWork(work *DistributeWork) {
	_, err := w.Pond.AddWork(DistributeEvent, work)
	if err != nil {
		w.Logger.Error("failed to add distribute work", zap.Error(err))
	}
}

// ---------------------------------------------------------------

// acceptClientWork accepts clients and do the following steps.
// 1. Accept a receivedStream.
// 2. Read client authentication token and topics.
// 3. Authenticate the client.
// 3.1 If the authentication is successful, opens sendStream for each topic and add them to eventSources.
// 3.2 If the authentication is not successful, closes the connection.
func (w *Worker) acceptClientWork(work any) any {
	runtime.LockOSThread()

	server, ok := work.(*Server)
	if !ok {
		return nil
	}

	for {
		background := context.Background()

		connection, err := server.Listener.Accept(background)
		if err != nil {
			w.Logger.Error("failed to accept new client", zap.Error(err))

			continue
		}

		w.Logger.Info("found a new client")

		go server.handleClient(connection)
	}
}

func (w *Worker) AddAcceptClientWork(server *Server, count int) {
	for range count {
		_, err := w.Pond.AddWork(AcceptClient, server)
		if err != nil {
			w.Logger.Error("failed to add accept client work", zap.Error(err))
		}
	}
}
