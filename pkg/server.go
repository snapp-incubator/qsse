package pkg

import (
	"crypto/tls"
	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/internal"
)

type Server interface {
	Publish(topic string, event []byte)

	SetAuthentication(handler func(token string) bool)
}

// NewServer creates a new server and listen for connections on the given address.
func NewServer(address string, tlsConfig *tls.Config, topics []string) (Server, error) {
	listener, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}

	server := internal.Server{
		Listener:     listener,
		Authenticate: internal.DefaultAuthenticationFunc,
		EventSources: make(map[string]*internal.EventSource),
	}

	server.GenerateEventSources(topics)

	go server.AcceptClients()

	return &server, nil
}