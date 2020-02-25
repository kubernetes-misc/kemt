package model

type Subscription struct {
	ID string
	r  *Rotary
}

func (s *Subscription) Close() {
	s.r.Lock()
	defer s.r.Unlock()
	delete(s.r.subscribers, s.ID)
}
