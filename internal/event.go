package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lucas-clemente/quic-go"
	"go.uber.org/atomic"
)

// EventSource is a struct for topic channel and its subscribers.
type EventSource struct {
	Topic                 string
	DataChannel           chan []byte
	Subscribers           []Subscriber
	IncomingSubscribers   chan Subscriber
	SubscriberWaitingList []Subscriber
	Metrics               Metrics
	Cleaning              *atomic.Bool
}

type Event struct {
	Topic string `json:"topic,omitempty"`
	Data  []byte `json:"data,omitempty"`
}

func NewEventSource(
	topic string,
	dataChannel chan []byte,
	subscribers []Subscriber,
	metric Metrics,
) *EventSource {
	return &EventSource{
		Topic:                 topic,
		DataChannel:           dataChannel,
		Subscribers:           subscribers,
		IncomingSubscribers:   make(chan Subscriber),
		SubscriberWaitingList: make([]Subscriber, 0),
		Metrics:               metric,
		Cleaning:              atomic.NewBool(false),
	}
}

func NewEvent(topic string, data []byte) *Event {
	return &Event{Topic: topic, Data: data}
}

// DistributeEvents distribute events from channel between subscribers.
func (e *EventSource) DistributeEvents(worker Worker) {
	for event := range e.DataChannel {
		work := NewSubscribeWork(event, e)
		worker.AddDistributeWork(work)
	}
}

func (e *EventSource) CleanCorruptSubscribers() {
	for _ = range time.Tick(1 * time.Second) {
		e.Cleaning.Store(true)

		i := 0

		for _, subscriber := range e.Subscribers {
			if !subscriber.Corrupt.Load() {
				e.Subscribers[i] = subscriber
				i++
			}
		}

		diff := len(e.Subscribers) - i
		if diff > 0 {
			log.Printf("cleaned %d corrupt subscribers\n", diff)
		}

		e.Subscribers = append(e.Subscribers[:i], e.SubscriberWaitingList...)
		e.SubscriberWaitingList = make([]Subscriber, 0)

		e.Cleaning.Store(false)
	}
}

func (e *EventSource) HandleNewSubscriber() {
	for subscriber := range e.IncomingSubscribers {
		if e.Cleaning.Load() {
			e.SubscriberWaitingList = append(e.SubscriberWaitingList, subscriber)
		} else {
			e.Subscribers = append(e.Subscribers, subscriber)
		}
	}
}

// WriteData writes data to stream.
func WriteData(data any, sendStream quic.SendStream) error {
	switch data := data.(type) {
	case []byte:
		if _, err := sendStream.Write(data); err != nil {
			return fmt.Errorf("write on stream failed %w", err)
		}
	default:
		bytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshaling data to json failed %w", err)
		}

		if _, err := sendStream.Write(bytes); err != nil {
			return fmt.Errorf("write on stream failed %w", err)
		}
	}

	if _, err := sendStream.Write([]byte{DELIMITER}); err != nil {
		return fmt.Errorf("write on stream failed %w", err)
	}

	return nil
}
