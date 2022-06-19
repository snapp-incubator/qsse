package qsse

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
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

func NewClient(address string, topics []string, config *ClientConfig) (Client, error) {
	processedConfig := processConfig(config)

	connection, err := quic.DialAddr(address, processedConfig.TLSConfig, nil)
	l := internal.NewLogger().Named("client")

	if err != nil {
		if config.ReconnectPolicy.Retry {
			l.Warn("Failed to connect to server, retrying...")

			c, res := reconnect(*config.ReconnectPolicy, address, processedConfig.TLSConfig, l.Named("reconnect"))
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
		log.Printf("failed to marshal offer: %+v\n", err)

		return nil, internal.ErrFailedToMarshal
	}

	stream, err := connection.OpenUniStream()
	if err != nil {
		log.Printf("failed to open send stream: %+v\n", err)
		internal.CloseClientConnection(connection, internal.CodeFailedToCreateStream, internal.ErrFailedToCreateStream)

		return nil, internal.ErrFailedToCreateStream
	}

	err = internal.WriteData(bytes, stream)
	if err != nil {
		log.Printf("failed to send offer to server: %+v\n", err)
		internal.CloseClientConnection(connection, internal.CodeFailedToSendOffer, internal.ErrFailedToSendOffer)

		return nil, internal.ErrFailedToSendOffer
	}

	_ = stream.Close()

	receiveStream, err := connection.AcceptUniStream(context.Background())
	if err != nil {
		log.Printf("failed to open receive stream: %+v\n", err)
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

		l.Warn("failed to reconnect", zap.Error(err))

		time.Sleep(time.Duration(policy.RetryInterval) * time.Millisecond)
	}

	return nil, false
}
