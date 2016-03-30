package crawler

import (
	"container/heap"
	"errors"
	"log"
	"net/url"
	"sync"
	"time"
)

// queue creates an infinite buffered channel. Queue receives input on push
// and sending output to pop. Queue should be run in its own goroutine. On
// termination queue closes pop. ttl defines the duration that a queue can
// wait without receiving new URLs to fetch. If the idle timeout is reached
// queue returns and closes pop.
func queue(push <-chan *url.URL, pop chan<- *url.URL, ttl time.Duration) {
	queue := make([]*url.URL, 0, 64)
	timer := time.NewTimer(ttl)
	defer func() {
		for len(queue) > 0 {
			pop <- queue[0]
			queue = queue[1:]
		}
		close(pop)
	}()

	for {
		if len(queue) == 0 {
			select {
			case url, ok := <-push:
				if !ok {
					return
				}
				queue = append(queue, url)
				timer.Reset(ttl)
			case <-timer.C:
				return
			}
		}

		select {
		case url, ok := <-push:
			if !ok {
				return
			}
			queue = append(queue, url)
			timer.Reset(ttl)
		case pop <- queue[0]:
			queue = queue[1:]
		case <-timer.C:
			return
		}
	}
}

type Producer struct {
	push   chan<- *url.URL
	mu     sync.RWMutex
	closed bool
}

func (p *Producer) Send(url *url.URL) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errors.New("producer is shut down")
	}
	log.Printf("producing %v", url)
	p.push <- url
	p.mu.Unlock()
	return nil
}

func (p *Producer) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errors.New("producer is shut down")
	}
	p.closed = true
	close(p.push)
	p.mu.Unlock()
	return nil
}

type Balancer struct {
	pop   <-chan *url.URL
	mu    sync.Mutex
	table map[int]int
	pool  pool

	wg   *sync.WaitGroup
	done chan struct{}
}

type Operation func(url *url.URL, prod *Producer)

func Start(op Operation, num int, ttl time.Duration) (*Balancer, *Producer) {
	push, pop := make(chan *url.URL, 64), make(chan *url.URL, 64)
	b := &Balancer{
		table: make(map[int]int, num),
		pool:  make(pool, 0, num),
		done:  make(chan struct{}),
		pop:   pop,
		wg:    &sync.WaitGroup{},
	}

	prod := &Producer{push: push}
	heap.Init(&b.pool)
	for i := 0; i < num; i++ {
		w := &worker{
			work:    make(chan *url.URL, 32),
			prod:    prod,
			pending: 0,
			id:      i,
			op:      op,
			wg:      b.wg,
		}
		heap.Push(&b.pool, w)
		b.table[i] = w.pos
		b.wg.Add(1)
		go w.start()
	}

	go queue(push, pop, ttl)
	go b.start()

	return b, prod
}

func (b *Balancer) Done() <-chan struct{} { return b.done }

func (b *Balancer) start() {
	for url := range b.pop {
		b.dispatch(url)
	}
	b.mu.Lock()
	for _, w := range b.pool {
		close(w.work)
	}
	b.mu.Unlock()
	b.wg.Wait()
	close(b.done)
}

func (b *Balancer) dispatch(url *url.URL) {
	b.mu.Lock()
	w := heap.Pop(&b.pool).(*worker)
	w.pending++
	w.work <- url
	heap.Push(&b.pool, w)
	b.table[w.id] = w.pos
	b.mu.Unlock()
}

func (b *Balancer) completed(id int) {
	b.mu.Lock()
	pos := b.table[id]
	w := b.pool[pos]
	w.pending--
	heap.Remove(&b.pool, pos)
	heap.Push(&b.pool, w)
	b.mu.Unlock()
}

type worker struct {
	op      func(url *url.URL, prod *Producer)
	prod    *Producer
	work    chan *url.URL
	wg      *sync.WaitGroup
	pending int
	pos     int
	id      int
}

func (w *worker) start() {
	for url := range w.work {
		log.Printf("worker%.4d received %s", w.id, url)
		w.op(url, w.prod)
	}
	w.prod.Close()
	w.wg.Done()
}

type pool []*worker

func (p pool) Len() int { return len(p) }

func (p pool) Less(i, j int) bool {
	return p[i].pending < p[j].pending
}

func (p *pool) Swap(i, j int) {
	a := *p
	a[i], a[j] = a[j], a[i]
	a[i].pos = i
	a[j].pos = j
}

func (p *pool) Push(x interface{}) {
	a := *p
	n := len(a)
	a = a[0 : n+1]
	w := x.(*worker)
	a[n] = w
	w.pos = n
	*p = a
}

func (p *pool) Pop() interface{} {
	a := *p
	*p = a[0 : len(a)-1]
	w := a[len(a)-1]
	w.pos = -1
	return w
}
