package tt

func NewSubscription(filter string, handlers ...PubHandler) *Subscription {
	r := &Subscription{
		TopicFilter: NewTopicFilter(filter),
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
