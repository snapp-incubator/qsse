package internal

import (
	"encoding/json"
	"github.com/lucas-clemente/quic-go"
	"log"
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
			err := WriteData(event, subscriber)
			if err != nil {
				log.Printf("err while sending event to client: %s", err.Error())
				receiver.Subscribers = append(receiver.Subscribers[:i], receiver.Subscribers[i+1:]...)
			}
		}
	}
}

// WriteData writes data to stream.
func WriteData(data any, sendStream quic.SendStream) error {
	var err error
	switch data.(type) {
	case []byte:
		_, err = sendStream.Write(data.([]byte))
	default:
		bytes, _ := json.Marshal(data)
		_, err = sendStream.Write(bytes)
	}

	if err != nil {
		return err
	}

	_, err = sendStream.Write([]byte{DELIMITER})

	return err
}
