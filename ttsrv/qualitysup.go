package ttsrv

import (
	"context"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

func NewQualitySupport(transmit tt.Handler) *QualitySupport {
	return &QualitySupport{
		transmit: transmit,
	}
}

// QualitySupport middleware which for now only supports QoS 0.
type QualitySupport struct {
	max      uint8
	transmit tt.Handler
}


func (s *QualitySupport) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Publish:
			// Disconnect any attempts to publish exceeding qos.
			// Specified in section 3.3.1.2 QoS
			if p.QoS() > s.max {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				return s.transmit(ctx, d)
			}
		}
		return next(ctx, p)
	}
}

func (s *QualitySupport) Out(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(s.max)
		}
		return next(ctx, p)
	}
}

// According to
// https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/
// the pubcomp packet is send only after a successful
// pubrel has was received. It feels like if the QoS 2 level is for udp like transports.
