package tt

// MustNewSubscription panics on bad filter
func MustNewSubscription(filter string, handlers ...PubHandler) *Subscription {
	tf, err := ParseFilterExpr(filter)
	if err != nil {
		panic(err.Error())
	}
	return NewSubscription(tf, handlers...)
}

func NewSubscription(filter *FilterExpr, handlers ...PubHandler) *Subscription {
	r := &Subscription{
		FilterExpr: filter,
		handlers:   handlers,
	}
	return r
}

type Subscription struct {
	*FilterExpr

	handlers []PubHandler
}

func (r *Subscription) String() string {
	return r.filter
}

func (r *Subscription) Filter() string { return r.filter }
