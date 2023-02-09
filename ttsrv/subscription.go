package ttsrv

import "github.com/gregoryv/tt"

// MustNewSubscription panics on bad filter
func MustNewSubscription(filter string, handlers ...PubHandler) *Subscription {
	tf, err := tt.ParseFilterExpr(filter)
	if err != nil {
		panic(err.Error())
	}
	return NewSubscription(tf, handlers...)
}

func NewSubscription(filter *tt.FilterExpr, handlers ...PubHandler) *Subscription {
	r := &Subscription{
		FilterExpr: filter,
		handlers:   handlers,
	}
	return r
}

type Subscription struct {
	*tt.FilterExpr

	handlers []PubHandler
}

func (r *Subscription) String() string {
	return r.Filter()
}
