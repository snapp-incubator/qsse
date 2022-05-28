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
)

const (
	reconnectRetryNumber   = 5
	reconnectRetryInterval = 5
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

func NewClient(address string, topics []string, config *ClientConfig) (Client, error) {
	processedConfig := processConfig(config)

	connection, err := quic.DialAddr(address, processedConfig.TLSConfig, nil)
	if err != nil {
		if config.ReconnectPolicy.Retry {
			log.Println("Failed to connect to server, retrying...")

			c, res := reconnect(*config.ReconnectPolicy, address, processedConfig.TLSConfig)
			if !res {
				log.Println("reconnecting failed")

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
	}

	offer := internal.NewOffer(processedConfig.Token, topics)
	bytes, _ := json.Marshal(offer) //nolint:errchkjson

	stream, _ := connection.OpenUniStream()

	_ = internal.WriteData(bytes, stream)

	_ = stream.Close()

	receiveStream, _ := connection.AcceptUniStream(context.Background())

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

func reconnect(policy ReconnectPolicy, address string, tlcCfg *tls.Config) (quic.Connection, bool) {
	for i := 0; i < policy.RetryTimes; i++ {
		connection, err := quic.DialAddr(address, tlcCfg, nil)
		if err == nil {
			return connection, true
		}

		log.Printf("failed to reconnect: %+v", err)

		time.Sleep(time.Duration(policy.RetryInterval) * time.Millisecond)
	}

	return nil, false
}
