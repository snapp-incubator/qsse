package internal

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
)

// EventSource is a struct for topic channel and its subscribers.
type EventSource struct {
	Topic       string
	DataChannel chan []byte
	Subscribers []quic.SendStream
}

type Event struct {
	Topic string `json:"topic,omitempty"`
	Data  []byte `json:"data,omitempty"`
}

func NewEventSource(topic string, dataChannel chan []byte, subscribers []quic.SendStream) *EventSource {
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
func WriteData(data any, sendStream quic.SendStream) error {
	var err error
	switch t := data.(type) {
	case []byte:
		_, err = sendStream.Write(t)
	default:
		bytes, _ := json.Marshal(data) //nolint:errchkjson
		_, err = sendStream.Write(bytes)
	}

	if err != nil {
		return errors.Errorf("failed to write data: %v", err)
	}

	_, err = sendStream.Write([]byte{DELIMITER})

	return err
}
