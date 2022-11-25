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
	transmit Handler
}

func (s *QualitySupport) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Publish:
			switch p.QoS() {
			case 1:
				a := mq.NewPubAck()
				a.SetPacketID(p.PacketID())
				return s.transmit(ctx, a)

			case 2:
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
			p.SetMaxQoS(1)
		}
		return next(ctx, p)
	}
}
