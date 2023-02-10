package ttsrv

import "github.com/gregoryv/tt"

// MustNewSubscription panics on bad filter
func MustNewSubscription(filter string, handlers ...PubHandler) *Subscription {
	tf, err := tt.ParseTopicFilter(filter)
	if err != nil {
		panic(err.Error())
	}
	return NewSubscription(tf, handlers...)
}

func NewSubscription(filter *tt.TopicFilter, handlers ...PubHandler) *Subscription {
	r := &Subscription{
		TopicFilter: filter,
		handlers:    handlers,
	}
	return r
}

type Subscription struct {
	*tt.TopicFilter

	handlers []PubHandler
}

func (r *Subscription) String() string {
	return r.Filter()
}
