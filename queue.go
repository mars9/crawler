package crawler

import (
	"net/url"
	"sync"
	"time"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrNotAbsoluteURL = Error("not an absolute url")
	ErrRejectedURL    = Error("url rejected")

	ErrQueueClosed  = Error("queue is shut down")
	ErrDuplicateURL = Error("duplicate url")
	ErrEmptyURL     = Error("empty url")
	ErrLimitReached = Error("limit reached")
)

type Pusher interface {
	Push(*url.URL) error
	Close() error
}

type Queue struct {
	push  chan *url.URL
	pop   chan *url.URL
	timer *time.Timer
	ttl   time.Duration

	mu     sync.Mutex
	closed bool
	set    map[string]struct{}
	limit  int64
	done   int64
}

func NewQueue(limit int64, ttl time.Duration) *Queue {
	q := &Queue{
		push:  make(chan *url.URL, 64), // queue channel capacity
		pop:   make(chan *url.URL, 64), // queue channel capacity
		timer: time.NewTimer(ttl),
		ttl:   ttl,
		set:   make(map[string]struct{}),
		limit: limit,
	}
	go q.run(256) // initial queue slice capacity
	return q
}

func (q *Queue) Push(url *url.URL) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	if q.limit > 0 && q.done > q.limit {
		q.mu.Unlock()
		return ErrLimitReached
	}

	key := normalizeKey(url)
	if len(key) == 0 {
		q.mu.Unlock()
		return ErrEmptyURL
	}
	if _, found := q.set[key]; found {
		q.mu.Unlock()
		return ErrDuplicateURL
	}

	q.set[key] = struct{}{}
	q.done++
	q.push <- url
	q.mu.Unlock()
	return nil
}

func (q *Queue) Close() error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.closed = true
	close(q.push)
	q.mu.Unlock()
	return nil
}

func (q *Queue) Pop() <-chan *url.URL {
	return q.pop
}

func (q *Queue) run(capacity int) {
	queue := make([]*url.URL, 0, capacity)
	defer func() {
		for len(queue) > 0 {
			q.pop <- queue[0]
			queue = queue[1:]
		}
		close(q.pop)
	}()

	for {
		if len(queue) == 0 {
			select {
			case url, ok := <-q.push:
				if !ok {
					q.Close()
					return
				}
				queue = append(queue, url)
				q.timer.Reset(q.ttl)
			case <-q.timer.C:
				q.Close()
				return
			}
		}

		select {
		case url, ok := <-q.push:
			if !ok {
				q.Close()
				return
			}
			queue = append(queue, url)
			q.timer.Reset(q.ttl)
		case q.pop <- queue[0]:
			queue = queue[1:]
		case <-q.timer.C:
			q.Close()
			return
		}
	}
}
