package internal

import (
	"bufio"
	"encoding/json"
	"log"

	quic "github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

type Client struct {
	Connection quic.Connection
	Token      string
	Topics     []string
	Logger     *zap.Logger
	Finder     Finder

	OnEvent   map[string]func(event []byte)
	OnMessage func(topic string, message []byte)
	OnError   func(code int, data map[string]any)
}

// DefaultOnMessage Default handler for processing incoming events without a handler.
var DefaultOnMessage = func(topic string, message []byte) { //nolint:gochecknoglobals
	log.Printf("topic: %s, message: %s\n", topic, string(message))
}

// DefaultOnError Default handler for processing errors.
// it listen to topic "error".
var DefaultOnError = func(code int, data map[string]any) { //nolint:gochecknoglobals
	log.Printf("code: %d, data: %v", code, data)
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
			c.Logger.Fatal("failed to read event", zap.Error(err))
		}

		var event Event
		if err = json.Unmarshal(bytes, &event); err != nil {
			c.Logger.Error("failed to unmarshal event", zap.Error(err))

			continue
		}

		switch {
		case event.Topic == ErrorTopic:
			err, e := UnmarshalError(event.Data)
			if e != nil {
				c.Logger.Error("error in unmarshalling", zap.Error(e))
			}

			c.OnError(err.Code, err.Data)
		default:
			topics := c.Finder.FindRelatedWildcardTopics(event.Topic, c.Topics)

			if len(topics) > 0 {
				for _, topic := range topics {
					eventHandler, ok := c.OnEvent[topic]
					if ok {
						eventHandler(event.Data)
					} else {
						c.OnMessage(topic, event.Data)
					}
				}
			} else {
				c.OnMessage(event.Topic, event.Data)
			}
		}
	}
}

// SetEventHandler sets the handler for the given topic.
func (c *Client) SetEventHandler(topic string, handler func([]byte)) {
	if IsSubscribeTopicValid(topic, c.Topics) {
		c.Topics = AppendIfMissing(c.Topics, topic)
		c.OnEvent[topic] = handler
	} else {
		c.Logger.Error("topic is not valid")
	}
}

// SetErrorHandler sets the handler for "error" topic.
func (c *Client) SetErrorHandler(handler func(code int, data map[string]any)) {
	c.OnError = handler
}

// SetMessageHandler sets the handler for all topics without handler and "error" topic.
func (c *Client) SetMessageHandler(handler func(topic string, message []byte)) {
	c.OnMessage = handler
}
