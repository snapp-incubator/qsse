package qsse

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"

	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/internal"
)

type Client interface {
	SetEventHandler(topic string, handler func([]byte))

	SetErrorHandler(handler func(code int, data map[string]any))

	SetMessageHandler(handler func(topic string, event []byte))
}

type ClientConfig struct {
	Token     string
	TLSConfig *tls.Config
}

func NewClient(address string, topics []string, config *ClientConfig) (Client, error) {
	processedConfig := processConfig(config)

	connection, err := quic.DialAddr(address, processedConfig.TLSConfig, nil)
	if err != nil {
		return nil, err
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
	bytes, _ := json.Marshal(offer)

	stream, _ := connection.OpenUniStream()

	internal.WriteData(bytes, stream)

	stream.Close()
  
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
		}
	}

	if config.TLSConfig == nil {
		config.TLSConfig = GetSimpleTLS()
	}

	return *config
}
