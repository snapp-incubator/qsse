package internal

import (
	"bufio"
	"encoding/json"
	"log"

	"github.com/lucas-clemente/quic-go"
)

type Client struct {
	Connection quic.Connection
	Token      string
	Topics     []string

	OnEvent   map[string]func(event []byte)
	OnMessage func(topic string, message []byte)
	OnError   func(code int, data map[string]any)
}

// DefaultOnMessage Default handler for processing incoming events without a handler.
var DefaultOnMessage = func(topic string, message []byte) { //nolint:gochecknoglobals
	log.Printf("topic: %s\ndata: %s\n", topic, string(message))
}

// DefaultOnError Default handler for processing errors.
// it listen to topic "error".
var DefaultOnError = func(code int, data map[string]any) { //nolint:gochecknoglobals
	log.Printf("Error: %d - %+v\n", code, data)
}

// AcceptEvents reads events from the stream and calls the proper handler.
// order of calling handlers is as follows:
// 1. OnError if topic is "error"
// 2. OnEvent[topic]
// 3. OnMessage.
func (c *Client) AcceptEvents(reader *bufio.Reader) {
	for {
		bytes, err := reader.ReadBytes(DELIMITER)
		if err != nil {
			log.Fatalf("failed to read event: %+v", err)
		}

		var event Event
		if err = json.Unmarshal(bytes, &event); err != nil {
			checkError(err)
		}

		switch {
		case event.Topic == ErrorTopic:
			err := UnmarshalError(event.Data)
			c.OnError(err.Code, err.Data)
		case c.OnEvent[event.Topic] != nil:
			topics := FindRelatedWildcardTopics(event.Topic, c.Topics)
			for _, topic := range topics {
				c.OnEvent[topic](event.Data)
			}
		default:
			c.OnMessage(event.Topic, event.Data)
		}
	}
}

// SetEventHandler sets the handler for the given topic.
func (c *Client) SetEventHandler(topic string, handler func([]byte)) {
	c.OnEvent[topic] = handler
}

// SetErrorHandler sets the handler for "error" topic.
func (c *Client) SetErrorHandler(handler func(code int, data map[string]any)) {
	c.OnError = handler
}

// SetMessageHandler sets the handler for all topics without handler and "error" topic.
func (c *Client) SetMessageHandler(handler func(topic string, message []byte)) {
	c.OnMessage = handler
}
