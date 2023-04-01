package qsse

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/auth"
	"github.com/snapp-incubator/qsse/internal"
)

const (
	DefCleaningInterval          = 10 * time.Second
	DefClientAcceptorCount       = 1
	DefClientAcceptorQueueSize   = 1
	DefEventDistributorCount     = 1
	DefEVentDistributorQueueSize = 10
)

type ServerConfig struct {
	Metric    *MetricConfig
	TLSConfig *tls.Config
	Worker    *WorkerConfig
}

type WorkerConfig struct {
	CleaningInterval          time.Duration
	ClientAcceptorCount       int64
	ClientAcceptorQueueSize   int
	EventDistributorCount     int64
	EventDistributorQueueSize int
}

type MetricConfig struct {
	Namespace string
	Subsystem string
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

	listener, err := quic.ListenAddr(address, config.TLSConfig, nil) //nolint:typecheck
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}

	workerConfig := internal.WorkerConfig{
		ClientAcceptorCount:       config.Worker.ClientAcceptorCount,
		ClientAcceptorQueueSize:   config.Worker.ClientAcceptorQueueSize,
		EventDistributorCount:     config.Worker.EventDistributorCount,
		EventDistributorQueueSize: config.Worker.EventDistributorQueueSize,
	}

	metric := internal.NewMetrics(config.Metric.Namespace, config.Metric.Subsystem)
	l := internal.NewLogger().Named("server")
	worker := internal.NewWorker(workerConfig, l.Named("worker"))
	server := internal.Server{
		Worker:        worker,
		Listener:      listener,
		Authenticator: auth.AuthenticatorFunc(internal.DefaultAuthenticationFunc),
		Authorizer:    auth.AuthorizerFunc(internal.DefaultAuthorizationFunc),
		EventSources:  make(map[string]*internal.EventSource),
		Topics:        topics,
		Metrics:       metric,
		Finder: internal.Finder{
			Logger: l.Named("finder"),
		},
		Logger:           l,
		CleaningInterval: config.Worker.CleaningInterval,
	}

	server.GenerateEventSources(topics)

	worker.AddAcceptClientWork(&server, int(config.Worker.ClientAcceptorCount))

	return &server, nil
}

func processServerConfig(cfg *ServerConfig) *ServerConfig {
	if cfg == nil {
		return &ServerConfig{
			Metric: &MetricConfig{
				Namespace: "qsse",
				Subsystem: "qsse",
			},
			TLSConfig: GetDefaultTLSConfig(),
			Worker: &WorkerConfig{
				CleaningInterval:          DefCleaningInterval,
				ClientAcceptorCount:       DefClientAcceptorCount,
				ClientAcceptorQueueSize:   DefClientAcceptorQueueSize,
				EventDistributorCount:     DefEventDistributorCount,
				EventDistributorQueueSize: DefEVentDistributorQueueSize,
			},
		}
	}

	if cfg.Metric == nil {
		cfg.Metric = &MetricConfig{
			Namespace: "qsse",
			Subsystem: "qsse",
		}
	}

	if cfg.TLSConfig == nil {
		cfg.TLSConfig = GetDefaultTLSConfig()
	}

	if cfg.Worker == nil {
		cfg.Worker = &WorkerConfig{
			CleaningInterval:          DefCleaningInterval,
			ClientAcceptorCount:       DefClientAcceptorCount,
			ClientAcceptorQueueSize:   DefClientAcceptorQueueSize,
			EventDistributorCount:     DefEventDistributorCount,
			EventDistributorQueueSize: DefEVentDistributorQueueSize,
		}
	}

	return cfg
}
