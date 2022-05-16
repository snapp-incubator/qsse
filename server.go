package qsse

import (
	"crypto/tls"
	"github.com/snapp-incubator/qsse/internal"
)

type Server interface {
	Publish(topic string, event []byte)

	SetAuthentication(handler func(token string) bool)
}

func NewServer(address string, tlsConfig *tls.Config, topics []string) (Server, error) {
	server, err := internal.NewServer(address, tlsConfig, topics)
	if err != nil {
		return nil, err
	}

	return Server(server), nil
}
