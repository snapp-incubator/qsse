package qsse

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/lucas-clemente/quic-go"
	"log"
)

type Client struct {
	connection quic.Connection
	token      string
	topics     []string

	onEvent   map[string]func(event []byte)
	onMessage func(topic string, message []byte)
	onError   func(code int, message error)
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

func NewClient(address string, token string, topics []string) (*Client, error) {
	connection, err := quic.DialAddr(address, GetSimpleTLS(), nil)
	if err != nil {
		return nil, err
	}

	client := Client{
		connection: connection,
		token:      token,
		topics:     topics,
		onEvent:    make(map[string]func([]byte)),
		onMessage:  DefaultOnMessage,
		onError:    DefaultOnError,
	}

	offer := NewOffer(token, topics)
	bytes, _ := json.Marshal(offer)

	stream, _ := connection.OpenUniStream()

	writeData(bytes, stream)
	stream.Close()

	connection.ConnectionState()

	receiveStream, err := connection.AcceptUniStream(context.Background())
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(receiveStream)
	go client.acceptEvents(reader)

	return &client, nil
}

// acceptEvents reads events from the stream and calls the proper handler.
// order of calling handlers is as follows:
// 1. onError if topic is "error"
// 2. onEvent[topic]
// 3. onMessage
func (c *Client) acceptEvents(reader *bufio.Reader) {
	for {
		bytes, err := reader.ReadBytes(DELIMITER)
		if err != nil {
			log.Fatal("failed to read event: %+v", err)
		}

		var event Event
		json.Unmarshal(bytes, &event)

		if event.Topic == ErrorTopic {
			err := UnmarshalError(event.Data)
			c.onError(err.Code, err.Err)
		} else if c.onEvent[event.Topic] != nil {
			c.onEvent[event.Topic](event.Data)
		} else {
			c.onMessage(event.Topic, event.Data)
		}
	}
}

// SetEventHandler sets the handler for the given topic.
func (c *Client) SetEventHandler(topic string, handler func([]byte)) {
	c.onEvent[topic] = handler
}

// SetErrorHandler sets the handler for "error" topic.
func (c *Client) SetErrorHandler(handler func(code int, err error)) {
	c.onError = handler
}

// SetMessageHandler sets the handler for all topics without handler and "error" topic.
func (c Client) SetMessageHandler(handler func(topic string, message []byte)) {
	c.onMessage = handler
}
