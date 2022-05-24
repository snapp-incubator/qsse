package qsse

import (
	"crypto/tls"

	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/internal"
	"github.com/snapp-incubator/qsse/pkg"
)

type Server interface {
	Publish(topic string, event []byte)

	SetAuthenticator(pkg.Autheticator)
	SetAuthenticatorFunc(pkg.AutheticatorFunc)

	SetAuthorizer(pkg.Authorizer)
	SetAuthorizerFunc(pkg.AuthorizerFunc)
}

// NewServer creates a new server and listen for connections on the given address.
func NewServer(address string, tlsConfig *tls.Config, topics []string) (Server, error) {
	listener, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}

	server := internal.Server{
		Worker:       internal.NewWorker(),
		Listener:     listener,
		Autheticator: pkg.AutheticatorFunc(internal.DefaultAuthenticationFunc),
		Authorizer:   pkg.AuthorizerFunc(internal.DefaultAuthorizationFunc),
		EventSources: make(map[string]*internal.EventSource),
	}

	server.GenerateEventSources(topics)

	go server.AcceptClients()

	return &server, nil
}
