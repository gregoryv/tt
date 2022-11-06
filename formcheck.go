package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

func CheckForm(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(interface{ WellFormed() error }); ok {
			// todo mq.WellFormed must return a specific error
			if err := p.WellFormed(); err != nil {
				return err
			}
		}
		return next(ctx, p)
	}
}
