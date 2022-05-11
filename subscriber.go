package qsse

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

func NewSubscriber(connection quic.Connection) *Subscriber {
	stream, err := connection.AcceptUniStream(context.Background())
	checkError(err)

	reader := bufio.NewReader(stream)
	bytes, err := reader.ReadBytes(DELIMITER)
	checkError(err)

	var offer Offer
	err = json.Unmarshal(bytes, &offer)
	checkError(err)

	return &Subscriber{
		connection: connection,
		Token:      offer.Token,
		Topics:     offer.Topics,
	}
}
