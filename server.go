package qsse

import (
	"crypto/tls"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/auth"
	"github.com/snapp-incubator/qsse/internal"
)

type ServerConfig struct {
	Metric    *MetricConfig
	TLSConfig *tls.Config
}

type MetricConfig struct {
	NameSpace string
}

type Server interface {
	Publish(topic string, event []byte)

	SetAuthenticator(auth.Authenticator)
	SetAuthenticatorFunc(auth.AuthenticatorFunc)

	SetAuthorizer(auth.Authorizer)
	SetAuthorizerFunc(auth.AuthorizerFunc)

	MetricHandler() http.Handler
}

// NewServer creates a new server and listen for connections on the given address.
func NewServer(address string, topics []string, config *ServerConfig) (Server, error) {
	config = processServerConfig(config)

	listener, err := quic.ListenAddr(address, config.TLSConfig, nil)
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}

	metric := internal.NewMetrics(config.Metric.NameSpace)
	l := internal.NewLogger().Named("server")
	server := internal.Server{
		Worker:        internal.NewWorker(),
		Listener:      listener,
		Authenticator: auth.AuthenticatorFunc(internal.DefaultAuthenticationFunc),
		Authorizer:    auth.AuthorizerFunc(internal.DefaultAuthorizationFunc),
		EventSources:  make(map[string]*internal.EventSource),
		Topics:        topics,
		Metrics:       metric,
		Logger:        l,
	}

	server.GenerateEventSources(topics)

	go server.AcceptClients()

	return &server, nil
}

func processServerConfig(cfg *ServerConfig) *ServerConfig {
	if cfg == nil {
		return &ServerConfig{
			Metric: &MetricConfig{
				NameSpace: "qsse",
			},
			TLSConfig: GetDefaultTLSConfig(),
		}
	}

	if cfg.Metric == nil {
		cfg.Metric = &MetricConfig{
			NameSpace: "qsse",
		}
	}

	if cfg.TLSConfig == nil {
		cfg.TLSConfig = GetDefaultTLSConfig()
	}

	return cfg
}
