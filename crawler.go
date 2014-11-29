// Package crawler provides a crawler implementation.
package crawler

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/context"
)

// Default options.
const (
	DefaultUserAgent       = "Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0"
	DefaultRobotsUserAgent = "Googlebot (crawlbot v1)"
	DefaultTimeToLive      = 2 * DefaultCrawlDelay
	DefaultCrawlDelay      = 3 * time.Second
)

// Crawler represents a crawler implementation.
type Crawler interface {
	// Fetch issues a GET to the specified URL and returns the response body
	// and an error if any.
	Fetch(url *url.URL) (rc io.ReadCloser, err error)

	// Parse is called when visiting a page. Parse receives a http response
	// body reader and should return an error, if any. Can be nil.
	Parse(url *url.URL, body []byte) (err error)

	// Domain returns the host to crawl.
	Domain() (domain *url.URL)

	// Accept can be used to control the crawler.
	Accept(url *url.URL) (ok bool)

	// Seeds returns the base URLs used to start crawling.
	Seeds() []*url.URL

	// MaxVisit returns the maximum number of pages visited before stopping
	// the crawl. Note that the Crawler will send its stop signal once this
	// number of visits is reached, but workers may be in the process of
	// visiting other pages, so when the crawling stops, the number of pages
	// visited will be at least MaxVisits, possibly more.
	MaxVisit() (max uint32)

	// Delay returns the time to wait between each request to the same host.
	// The delay starts as soon as the response is received from the host.
	Delay() (delay time.Duration)

	// TTL returns the duration that a crawler goroutine can wait without
	// receiving new commands to fetch. If the idle time-to-live is reached,
	// the crawler goroutine is stopped and its resources are released. This
	// can be especially useful for long-running crawlers.
	TTL() (timeout time.Duration)
}

type worker struct {
	workerc chan<- chan *url.URL
	workc   chan *url.URL
	uid     uint8
	wg      *sync.WaitGroup
	client  *http.Client
	crawler Crawler
}

func (w *worker) Start(ctx context.Context, push chan<- *url.URL) {
	defer w.wg.Done()

	var err error
	for {
		w.workerc <- w.workc

		select {
		case url := <-w.workc:
			if err = Fetch(url, w.crawler, push); err != nil {
				break
			}
		case <-ctx.Done():
			return
		}

		if w.crawler.Delay() > 0 {
			time.Sleep(w.crawler.Delay())
		}
	}
}

// Start starts a new crawl. Crawlers defines the number concurrently
// working crawlers.
func Start(ctx context.Context, c Crawler, crawlers uint8) {
	canceler := make([]context.CancelFunc, crawlers)
	workerc := make(chan chan *url.URL, crawlers)
	workers := make([]*worker, crawlers)
	pushc := make(chan *url.URL, 16)
	popc := make(chan *url.URL, 16)
	wg := &sync.WaitGroup{}

	go Queue(pushc, popc)
	defer close(pushc)
	go func() {
		for _, seed := range c.Seeds() {
			pushc <- seed
		}
	}()

	for i := uint8(0); i < crawlers; i++ {
		workers[i] = &worker{
			workc:   make(chan *url.URL, 1),
			workerc: workerc,
			uid:     i,
			wg:      wg,
			client:  &http.Client{},
			crawler: c,
		}

		context, cancel := context.WithCancel(ctx)
		canceler[i] = cancel
		wg.Add(1)
		go workers[i].Start(context, pushc)
	}

	timer := time.NewTimer(c.TTL())
	donec := make(chan struct{}, 1)
	go func(popc <-chan *url.URL) {
		var visited uint32
		for url := range popc {
			visited++
			if c.MaxVisit() > 0 && c.MaxVisit() < visited {
				donec <- struct{}{}
				return
			}

			workc := <-workerc
			workc <- url

			timer.Reset(c.TTL())
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
