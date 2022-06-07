package qsse

import (
	"crypto/tls"

	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/auth"
	"github.com/snapp-incubator/qsse/internal"
)

type ServerConfig struct {
	NameSpace string
	Port      string
}

type Server interface {
	Publish(topic string, event []byte)

	SetAuthenticator(auth.Authenticator)
	SetAuthenticatorFunc(auth.AuthenticatorFunc)

	SetAuthorizer(auth.Authorizer)
	SetAuthorizerFunc(auth.AuthorizerFunc)
}

// NewServer creates a new server and listen for connections on the given address.
func NewServer(address string, tlsConfig *tls.Config, topics []string, config *ServerConfig) (Server, error) {
	listener, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}
	metric := internal.NewMetrics(config.NameSpace, config.Port)
	server := internal.Server{
		Worker:        internal.NewWorker(),
		Listener:      listener,
		Authenticator: auth.AuthenticatorFunc(internal.DefaultAuthenticationFunc),
		Authorizer:    auth.AuthorizerFunc(internal.DefaultAuthorizationFunc),
		EventSources:  make(map[string]*internal.EventSource),
		Topics:        topics,
		Metrics:       metric,
	}

	server.GenerateEventSources(topics)

	go server.AcceptClients()

	return &server, nil
}
