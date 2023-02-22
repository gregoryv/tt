package tt

import "context"

type Event int

const (
	EventUndefined Event = iota
	EventClientUp

	EventServerUp
)

type EventHandler func(context.Context, Event)
