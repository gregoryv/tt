package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

func NewQualitySupport(transmit Handler) *QualitySupport {
	return &QualitySupport{
		transmit: transmit,
	}
}

type QualitySupport struct {
	max      uint8
	transmit Handler
}

func (s *QualitySupport) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Publish:
			if p.QoS() > s.max {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				return s.transmit(ctx, d)
			}
		}
		return next(ctx, p)
	}
}

func (s *QualitySupport) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(s.max)
		}
		return next(ctx, p)
	}
}
