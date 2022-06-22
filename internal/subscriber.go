package internal

import (
	"github.com/lucas-clemente/quic-go"
	"go.uber.org/atomic"
)

type Subscriber struct {
	Stream  quic.SendStream
	Corrupt *atomic.Bool
}

func NewSubscriber(stream quic.SendStream) Subscriber {
	return Subscriber{
		Stream:  stream,
		Corrupt: atomic.NewBool(false),
	}
}
