package tt

import "sync/atomic"

func NewServerStats() *ServerStats {
	return &ServerStats{}
}

type ServerStats struct {
	ConnCount  int64
	ConnActive int64
}

func (s *ServerStats) AddConn() {
	atomic.AddInt64(&s.ConnCount, 1)
	atomic.AddInt64(&s.ConnActive, 1)
}

func (s *ServerStats) RemoveConn() {
	atomic.AddInt64(&s.ConnActive, -1)
}
