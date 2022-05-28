package internal

import (
	"encoding/json"
	"fmt"

	"github.com/lucas-clemente/quic-go" //nolint:typecheck
)

// EventSource is a struct for topic channel and its subscribers.
type EventSource struct {
	Topic       string
	DataChannel chan []byte
	Subscribers []quic.SendStream //nolint:typecheck
}

type Event struct {
	Topic string `json:"topic,omitempty"`
	Data  []byte `json:"data,omitempty"`
}

func NewEventSource(topic string, dataChannel chan []byte, subscribers []quic.SendStream) *EventSource { //nolint:typecheck
	return &EventSource{Topic: topic, DataChannel: dataChannel, Subscribers: subscribers}
}

func NewEvent(topic string, data []byte) *Event {
	return &Event{Topic: topic, Data: data}
}

// TransferEvents distribute events from channel between subscribers.
func (receiver *EventSource) TransferEvents(worker Worker) {
	for event := range receiver.DataChannel {
		work := NewSubscribeWork(event, receiver)
		worker.SubscribePool.Process(work)
	}
}

// WriteData writes data to stream.
func WriteData(data any, sendStream quic.SendStream) error { //nolint:typecheck
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
