// Copyright 2016 Markus Sonderegger. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crawler

import "net/url"

// NewQueue returns a infinite buffered channel queue. A queue receives
// input on the send channel und sending output to the receive channel. On
// termination queue closes the receive channel. The caller must close the
// send channel.
func NewQueue(capacity int) (chan<- *url.URL, <-chan *url.URL) {
	push := make(chan *url.URL)
	pop := make(chan *url.URL)
	go queue(push, pop)
	return push, pop
}

// queue creates an infinite buffered channel. Queue receives input on
// push and sending output to pop. Queue should be run in its own
// goroutine. On termination queue closes pop.
func queue(push <-chan *url.URL, pop chan<- *url.URL) {
	queue := make([]*url.URL, 0, 64)
	defer func() {
		for len(queue) > 0 {
			pop <- queue[0]
			queue = queue[1:]
		}
		close(pop)
	}()

	for {
		if len(queue) == 0 {
			url, ok := <-push
			if !ok {
				return
			}
			queue = append(queue, url)
		}

		select {
		case url, ok := <-push:
			if !ok {
				return
			}
			queue = append(queue, url)

		case pop <- queue[0]:
			queue = queue[1:]
		}
	}
}
