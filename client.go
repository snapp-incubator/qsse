package qsse

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/internal"
	"go.uber.org/zap"
)

const (
	reconnectRetryNumber   = 5
	reconnectRetryInterval = 5
)

type Client interface {
	SetEventHandler(topic string, handler func([]byte))

	SetErrorHandler(handler func(code int, data map[string]any, l *zap.Logger))

	SetMessageHandler(handler func(topic string, event []byte, l *zap.Logger))
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
	l := internal.NewLogger()

	connection, err := quic.DialAddr(address, processedConfig.TLSConfig, nil)
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

				return nil, err //nolint:wrapcheck
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
		OnEvent:    make(map[string]func([]byte)),
		OnMessage:  internal.DefaultOnMessage,
		OnError:    internal.DefaultOnError,
		Logger:     l,
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
		internal.CloseClientConnection(connection, internal.CodeFailedToCreateStream, internal.ErrFailedToCreateStream)

		return nil, internal.ErrFailedToCreateStream
	}

	err = internal.WriteData(bytes, stream)
	if err != nil {
		l.Error("failed to send offer to server", zap.Error(err))
		internal.CloseClientConnection(connection, internal.CodeFailedToSendOffer, internal.ErrFailedToSendOffer)

		return nil, internal.ErrFailedToSendOffer
	}

	_ = stream.Close()

	receiveStream, err := connection.AcceptUniStream(context.Background())
	if err != nil {
		l.Error("failed to open receive stream", zap.Error(err))
		internal.CloseClientConnection(connection, internal.CodeFailedToCreateStream, internal.ErrFailedToCreateStream)

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

func reconnect(policy ReconnectPolicy, address string, tlcCfg *tls.Config, l *zap.Logger) (quic.Connection, bool) {
	for i := 0; i < policy.RetryTimes; i++ {
		connection, err := quic.DialAddr(address, tlcCfg, nil)
		if err == nil {
			return connection, true
		}

		l.Error("failed to reconnect", zap.Error(err))

		time.Sleep(time.Duration(policy.RetryInterval) * time.Millisecond)
	}

	return nil, false
}
