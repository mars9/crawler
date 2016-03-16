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

// Start starts a new crawl. Crawlers defines the number concurrently
// working crawlers.
func Start(ctx context.Context, crawler Crawler, crawlers int) {
	canceler := make([]context.CancelFunc, crawlers)
	workerc := make(chan chan *url.URL, crawlers)
	workers := make([]*worker, crawlers)
	pushc := make(chan *url.URL, 16)
	popc := make(chan *url.URL, 16)
	wg := &sync.WaitGroup{}

	go Queue(pushc, popc)
	defer close(pushc)

	go func() {
		for _, seed := range crawler.Seeds() {
			pushc <- seed
		}
	}()

	for i := 0; i < crawlers; i++ {
		workers[i] = &worker{
			workc:   make(chan *url.URL, 1),
			workerc: workerc,
			id:      i,
			wg:      wg,
			c:       crawler,
		}

		context, cancel := context.WithCancel(ctx)
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
	case <-timer.C:
		for i := range canceler {
			canceler[i]()
		}
	case <-donec:
		for i := range canceler {
			canceler[i]()
		}
	case <-ctx.Done():
	}
	wg.Wait()
}
