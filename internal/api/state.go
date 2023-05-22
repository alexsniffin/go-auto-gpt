package api

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

type requestsCache struct {
	ids map[uuid.UUID]*actor.PID
}

// todo this should be persistent and thread safe
func newRequestsCache() *requestsCache {
	return &requestsCache{
		ids: map[uuid.UUID]*actor.PID{},
	}
}

func (s *requestsCache) remove(id uuid.UUID) {
	delete(s.ids, id)
}

func (s *requestsCache) add(id uuid.UUID, pid *actor.PID) {
	s.ids[id] = pid
}

func (s *requestsCache) get(id uuid.UUID) (*actor.PID, bool) {
	pid, ok := s.ids[id]
	return pid, ok
}
