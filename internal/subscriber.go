package internal

import (
	"bufio"
	"context"
	"encoding/json"

	"github.com/lucas-clemente/quic-go"
	"go.uber.org/zap"
)

type Subscriber struct {
	connection quic.Connection
	Token      string
	Topics     []string
}

type Offer struct {
	Token  string   `json:"token,omitempty"`
	Topics []string `json:"topics,omitempty"`
}

func NewOffer(token string, topics []string) Offer {
	return Offer{Token: token, Topics: topics}
}

func NewSubscriber(connection quic.Connection, logger *zap.Logger) *Subscriber {
	stream, err := connection.AcceptUniStream(context.Background())
	checkError(err, logger)

	reader := bufio.NewReader(stream)
	bytes, err := reader.ReadBytes(DELIMITER)
	checkError(err, logger)

	var offer Offer
	err = json.Unmarshal(bytes, &offer)
	checkError(err, logger)

	return &Subscriber{
		connection: connection,
		Token:      offer.Token,
		Topics:     offer.Topics,
	}
}
