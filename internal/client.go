package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lucas-clemente/quic-go"
	"go.uber.org/zap"
)

type Client struct {
	Connection quic.Connection
	Token      string
	Topics     []string
	Logger     *zap.Logger

	OnEvent   map[string]func(event []byte)
	OnMessage func(topic string, message []byte, l *zap.Logger)
	OnError   func(code int, data map[string]any, l *zap.Logger)
}

// DefaultOnMessage Default handler for processing incoming events without a handler.
var DefaultOnMessage = func(topic string, message []byte, l *zap.Logger) { //nolint:gochecknoglobals
	l.Info("default message", zap.String("topic", topic), zap.String("data", string(message)))
}

// DefaultOnError Default handler for processing errors.
// it listen to topic "error".
var DefaultOnError = func(code int, data map[string]any, l *zap.Logger) { //nolint:gochecknoglobals
	b := new(bytes.Buffer)
	for key, value := range data {
		if _, err := fmt.Fprintf(b, "%s=\"%s\"\n", key, value); err != nil {
			l.Warn("cannot parse data map", zap.Error(err))
		}
	}
	l.Info("default error message", zap.Int("code", code), zap.String("data", b.String()))
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
			checkError(err, c.Logger)
		}

		switch {
		case event.Topic == ErrorTopic:
			err := UnmarshalError(event.Data, c.Logger)
			c.OnError(err.Code, err.Data, c.Logger)
		default:
			topics := FindRelatedWildcardTopics(event.Topic, c.Topics, c.Logger)
			c.Logger.Info("events:", zap.String("events", strings.Join(topics, " ")))

			if len(topics) > 0 {
				for _, topic := range topics {
					eventHandler, ok := c.OnEvent[topic]
					if ok {
						eventHandler(event.Data)
					} else {
						c.OnMessage(topic, event.Data, c.Logger)
					}
				}
			} else {
				c.OnMessage(event.Topic, event.Data, c.Logger)
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
		c.Logger.Warn("topic is not valid")
	}
}

// SetErrorHandler sets the handler for "error" topic.
func (c *Client) SetErrorHandler(handler func(code int, data map[string]any, l *zap.Logger)) {
	c.OnError = handler
}

// SetMessageHandler sets the handler for all topics without handler and "error" topic.
func (c *Client) SetMessageHandler(handler func(topic string, message []byte, l *zap.Logger)) {
	c.OnMessage = handler
}
