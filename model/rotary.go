package model

import (
	"github.com/google/uuid"
	"sync"
)

func NewRotary(size int) *Rotary {
	result := &Rotary{
		size:        size,
		subscribers: map[string]chan string{},
	}
	return result
}

type Rotary struct {
	sync.Mutex
	size        int
	total       int
	start       *Log
	end         *Log
	subscribers map[string]chan string
}

func (r *Rotary) Add(s string) {
	defer r.publish(s)
	r.Lock()
	defer r.Unlock()
	if r.start == nil {
		r.start = &Log{
			body: s,
			next: nil,
		}
		r.total = 1
		return
	}
	prevEnd := r.end
	newEnd := &Log{
		body: s,
		next: nil,
	}
	if prevEnd != nil {
		prevEnd.next = newEnd
	} else {
		r.start.next = newEnd
	}
	r.end = newEnd
	r.total++

	if r.total > r.size {
		newHead := r.start.next
		r.start = newHead
		r.total--
	}

}

func (r *Rotary) ForEach(f func(Log)) {
	if r.start == nil {
		return
	}
	current := r.start
	for current != nil {
		f(*current)
		current = current.next
	}
}

func (r *Rotary) Subscribe(toSubscriber chan string) (s Subscription) {
	r.Lock()
	defer r.Unlock()
	s = Subscription{ID: uuid.New().String()}
	r.subscribers[s.ID] = toSubscriber
	return
}

func (r *Rotary) publish(s string) {
	for _, ch := range r.subscribers {
		ch <- s
	}
}
