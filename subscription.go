package tt

// MustNewSubscription panics on bad filter
func MustNewSubscription(filter string, handlers ...PubHandler) *Subscription {
	tf, err := ParseTopicFilter(filter)
	if err != nil {
		panic(err.Error())
	}
	return NewSubscription(tf, handlers...)
}

func NewSubscription(filter *TopicFilter, handlers ...PubHandler) *Subscription {
	r := &Subscription{
		TopicFilter: filter,
		handlers:    handlers,
	}
	return r
}

type Subscription struct {
	*TopicFilter

	handlers []PubHandler
}

func (r *Subscription) String() string {
	return r.filter
}

func (r *Subscription) Filter() string { return r.filter }
