package tt

type Event int

const (
	EventUndefined Event = iota
	EventClientUp
	EventClientDown

	EventServerUp
)
