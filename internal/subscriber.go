package internal

import (
	"bufio"
	"context"
	"encoding/json"

	"github.com/lucas-clemente/quic-go"
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

func NewSubscriber(connection quic.Connection) (*Subscriber, error) {
	stream, err := connection.AcceptUniStream(context.Background())
	if err != nil {
		return nil, ErrFailedToCreateStream
	}

	reader := bufio.NewReader(stream)

	bytes, err := reader.ReadBytes(DELIMITER)
	if err != nil {
		return nil, ErrFailedToReadOffer
	}

	var offer Offer
	if err := json.Unmarshal(bytes, &offer); err != nil {
		return nil, ErrFailedToUnmarshal
	}

	return &Subscriber{
		connection: connection,
		Token:      offer.Token,
		Topics:     offer.Topics,
	}, nil
}
