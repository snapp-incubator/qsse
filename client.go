package qsse

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	quic "github.com/quic-go/quic-go"
	"github.com/snapp-incubator/qsse/internal"
	"go.uber.org/zap"
)

const (
	reconnectRetryNumber   = 5
	reconnectRetryInterval = 5000
)

type Client interface {
	SetEventHandler(topic string, handler func([]byte))

	SetErrorHandler(handler func(code int, data map[string]any))

	SetMessageHandler(handler func(topic string, event []byte))
}

type ClientConfig struct {
	Token           string
	TLSConfig       *tls.Config
	ReconnectPolicy *ReconnectPolicy
}

type ReconnectPolicy struct {
	Retry         bool
	RetryTimes    int
	RetryInterval int // duration between retry intervals in milliseconds
}

//nolint:funlen
func NewClient(address string, topics []string, config *ClientConfig) (Client, error) {
	processedConfig := processConfig(config)
	l := internal.NewLogger().Named("client")

	connection, err := quic.DialAddr(context.Background(), address, processedConfig.TLSConfig, nil)
	if err != nil {
		if processedConfig.ReconnectPolicy.Retry {
			l.Warn("Failed to connect to server, retrying...")

			c, res := reconnect(
				*processedConfig.ReconnectPolicy,
				address,
				processedConfig.TLSConfig,
				l.Named("reconnect"),
			)
			if !res {
				l.Warn("reconnecting failed")

				return nil, err
			}

			connection = c
		} else {
			return nil, err //nolint:wrapcheck
		}
	}

	client := internal.Client{
		Connection: connection,
		Token:      processedConfig.Token,
		Topics:     topics,
		Finder: internal.Finder{
			Logger: l.Named("finder"),
		},
		OnEvent:   make(map[string]func([]byte)),
		OnMessage: internal.DefaultOnMessage,
		OnError:   internal.DefaultOnError,
		Logger:    l.Named("client"),
	}

	offer := internal.NewOffer(processedConfig.Token, topics)

	bytes, err := json.Marshal(offer)
	if err != nil {
		l.Error("failed to marshal offer", zap.Error(err))

		return nil, internal.ErrFailedToMarshal
	}

	stream, err := connection.OpenUniStream()
	if err != nil {
		l.Error("failed to open send stream", zap.Error(err))

		err = internal.CloseClientConnection(
			connection,
			internal.CodeFailedToCreateStream,
			internal.ErrFailedToCreateStream,
		)
		if err != nil {
			l.Error("failed to close client connection", zap.Error(err))
		}

		return nil, internal.ErrFailedToCreateStream
	}

	err = internal.WriteData(bytes, stream)
	if err != nil {
		l.Error("failed to send offer to server", zap.Error(err))

		err = internal.CloseClientConnection(
			connection,
			internal.CodeFailedToSendOffer,
			internal.ErrFailedToSendOffer,
		)
		if err != nil {
			l.Error("failed to close client connection", zap.Error(err))
		}

		return nil, internal.ErrFailedToSendOffer
	}

	_ = stream.Close()

	receiveStream, err := connection.AcceptUniStream(context.Background())
	if err != nil {
		l.Error("failed to open receive stream", zap.Error(err))

		err = internal.CloseClientConnection(
			connection,
			internal.CodeFailedToCreateStream,
			internal.ErrFailedToCreateStream,
		)
		if err != nil {
			l.Error("failed to close client connection", zap.Error(err))
		}

		return nil, internal.ErrFailedToCreateStream
	}

	reader := bufio.NewReader(receiveStream)
	go client.AcceptEvents(reader)

	return &client, nil
}

func processConfig(config *ClientConfig) ClientConfig {
	if config == nil {
		return ClientConfig{
			Token:     "",
			TLSConfig: GetSimpleTLS(),
			ReconnectPolicy: &ReconnectPolicy{
				Retry:         false,
				RetryTimes:    reconnectRetryNumber,
				RetryInterval: reconnectRetryInterval,
			},
		}
	}

	if config.TLSConfig == nil {
		config.TLSConfig = GetSimpleTLS()
	}

	if config.ReconnectPolicy == nil {
		config.ReconnectPolicy = &ReconnectPolicy{
			Retry:         false,
			RetryTimes:    reconnectRetryNumber,
			RetryInterval: reconnectRetryInterval,
		}
	}

	return *config
}

//nolint:typecheck
func reconnect(policy ReconnectPolicy, address string, tlcCfg *tls.Config, l *zap.Logger) (quic.Connection, bool) {
	for range policy.RetryTimes {
		connection, err := quic.DialAddr(context.Background(), address, tlcCfg, nil)
		if err == nil {
			return connection, true
		}

		l.Error("failed to reconnect", zap.Error(err))

		time.Sleep(time.Duration(policy.RetryInterval) * time.Millisecond)
	}

	return nil, false
}
