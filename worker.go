// Copyright 2016 Markus Sonderegger. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crawler

import (
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/context"
)

type worker struct {
	workerc chan<- chan *url.URL
	workc   chan *url.URL
	id      int
	wg      *sync.WaitGroup
	c       Crawler
}

func (w *worker) Start(ctx context.Context, push chan<- *url.URL) {
	defer w.wg.Done()

	var err error
	for {
		w.workerc <- w.workc

		select {
		case url := <-w.workc:
			if err = Fetch(url, w.c, push); err != nil {
				break
			}
		case <-ctx.Done():
			return
		}

		if w.c.Delay() > 0 {
			time.Sleep(w.c.Delay())
		}
	}
}

type Options func(*Option)

type Option struct {
	capacity int
	worker   int
}

func newOption(opts ...Options) *Option {
	o := &Option{}
	for _, opt := range opts {
		opt(o)
	}
	// TODO: use constants
	if o.capacity <= 0 {
		o.capacity = 64
	}
	if o.worker <= 0 {
		o.worker = 8
	}
	return o
}

func CrawlWorkers(num int) Options {
	return func(opt *Option) {
		opt.worker = num
	}
}

func QueueCapacity(num int) Options {
	return func(opt *Option) {
		opt.capacity = num
	}
}

// Start starts a new crawl. Crawlers defines the number concurrently
// working crawlers.
func Start(ctx context.Context, crawler Crawler, opts ...Options) {
	opt := newOption(opts...)
	canceler := make([]context.CancelFunc, opt.worker)
	workerc := make(chan chan *url.URL, opt.worker)
	workers := make([]*worker, opt.worker)
	wg := &sync.WaitGroup{}

	pushc, popc := NewQueue(opt.capacity)
	defer close(pushc)

	go func() {
		for _, seed := range crawler.Seeds() {
			pushc <- seed
		}
	}()

	for i := 0; i < opt.worker; i++ {
		workers[i] = &worker{
			workc:   make(chan *url.URL, 1),
			workerc: workerc,
			id:      i,
			wg:      wg,
			c:       crawler,
		}

		context, cancel := context.WithCancel(context.Background())
		canceler[i] = cancel
		wg.Add(1)
		go workers[i].Start(context, pushc)
	}

	timer := time.NewTimer(crawler.TTL())
	donec := make(chan struct{}, 1)
	go func(popc <-chan *url.URL) {
		var visited int64
		for url := range popc {
			visited++
			if crawler.MaxVisit() > 0 && crawler.MaxVisit() < visited {
				donec <- struct{}{}
				return
			}

			workc := <-workerc
			workc <- url

			timer.Reset(crawler.TTL())
		}
	}(popc)

	select {
	case <-ctx.Done():
	case <-timer.C:
	case <-donec:
	}
	for i := range canceler {
		canceler[i]()
	}

	wg.Wait()
}
