package internal

import (
	"github.com/lucas-clemente/quic-go"
)

type Subscriber struct {
	connection quic.Connection
	Token      string
	Topics     []string
}
