package crawler

import "net/url"

/*
// pushQueue pushes a new URL into the queue, expanding the queue to
// guarantee space for more URLs.
func pushQueue(q *[]*url.URL, link *url.URL) {
	if len(*q) == cap(*q) {
		nq := make([]*url.URL, len(*q), cap(*q)*2+len(*q))
		copy(nq, *q)
		*q = nq
	}
	*q = append(*q, link)
}

// getQueue pops out the first URL from the queue. getQueue panics if
// there is no such element.
func getQueue(q *[]*url.URL) *url.URL {
	return (*q)[0]
}

// delQueue deletes the first URL from the queue. delQueue panics if
// there is no such element.
func delQueue(q *[]*url.URL) {
	*q = (*q)[1:]
}
*/

// Queue creates an infinite buffered channel. Queue receives input on
// push and sending output to pop. Queue should be run in its own
// goroutine. On termination Queue closes pop.
func Queue(push <-chan *url.URL, pop chan<- *url.URL) {
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
