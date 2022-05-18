package internal

import (
	"encoding/json"
	"fmt"
	"log"

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

// transferEvents distribute events from channel between subscribers.
func (receiver *EventSource) transferEvents() {
	for event := range receiver.DataChannel {
		log.Println("Number of Subscribers:", len(receiver.Subscribers))

		for i, subscriber := range receiver.Subscribers {
			log.Println("Sending event to subscriber for topic:", receiver.Topic)
			event := NewEvent(receiver.Topic, event)

			if err := WriteData(event, subscriber); err != nil {
				log.Printf("err while sending event to client: %s", err.Error())

				receiver.Subscribers = append(receiver.Subscribers[:i], receiver.Subscribers[i+1:]...)
			}
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
