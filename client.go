package qsse

import "github.com/snapp-incubator/qsse/internal"

type Client interface {
	SetEventHandler(topic string, handler func([]byte))

	SetErrorHandler(handler func(code int, err error))

	SetMessageHandler(handler func(topic string, event []byte))
}

func NewClient(address string, token string, topics []string) (Client, error) {
	client, err := internal.NewClient(address, token, topics, GetSimpleTLS())
	if err != nil {
		return nil, err
	}

	return Client(client), nil
}
