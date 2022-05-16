package internal

import (
	"bufio"
	"encoding/json"
	"github.com/lucas-clemente/quic-go"
	"log"
)

type Client struct {
	Connection quic.Connection
	Token      string
	Topics     []string

	OnEvent   map[string]func(event []byte)
	OnMessage func(topic string, message []byte)
	OnError   func(code int, message error)
}

// DefaultOnMessage Default handler for processing incoming events without a handler.
var DefaultOnMessage = func(topic string, message []byte) {
	log.Printf("topic: %s\ndata: %s\n", topic, string(message))
}

// DefaultOnError Default handler for processing errors.
// it listen to topic "error"
var DefaultOnError = func(code int, message error) {
	log.Printf("Error: %d - %+v\n", code, message)
}

// AcceptEvents reads events from the stream and calls the proper handler.
// order of calling handlers is as follows:
// 1. OnError if topic is "error"
// 2. OnEvent[topic]
// 3. OnMessage
func (c *Client) AcceptEvents(reader *bufio.Reader) {
	for {
		bytes, err := reader.ReadBytes(DELIMITER)
		if err != nil {
			log.Fatalf("failed to read event: %+v", err)
		}

		var event Event
		json.Unmarshal(bytes, &event)

		if event.Topic == ErrorTopic {
			err := UnmarshalError(event.Data)
			c.OnError(err.Code, err.Err)
		} else if c.OnEvent[event.Topic] != nil {
			c.OnEvent[event.Topic](event.Data)
		} else {
			c.OnMessage(event.Topic, event.Data)
		}
	}
}

// SetEventHandler sets the handler for the given topic.
func (c *Client) SetEventHandler(topic string, handler func([]byte)) {
	c.OnEvent[topic] = handler
}

// SetErrorHandler sets the handler for "error" topic.
func (c *Client) SetErrorHandler(handler func(code int, err error)) {
	c.OnError = handler
}

// SetMessageHandler sets the handler for all topics without handler and "error" topic.
func (c Client) SetMessageHandler(handler func(topic string, message []byte)) {
	c.OnMessage = handler
}
