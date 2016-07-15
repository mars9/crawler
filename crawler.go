package crawler

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/mars9/crawler/sitemap"
	"github.com/mars9/crawler/transform"
	"golang.org/x/net/html"
)

const (
	DefaultUserAgent   = "Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0"
	DefaultRobotsAgent = "Googlebot (crawlbot v1)"
	DefaultTimeToLive  = 3 * DefaultDelay
	DefaultDelay       = 3 * time.Second
)

func init() { log.SetFlags(log.Ldate | log.Lmicroseconds | log.LUTC) }

func Get(url *url.URL, agent string, robots Robots) (io.ReadCloser, error) {
	if !url.IsAbs() {
		return nil, errors.New("not an absolute URL")
	}
	if robots != nil && !robots.Test(url) {
		return nil, errors.New("rejected by robots.txt")
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if len(agent) == 0 {
		req.Header.Set("User-Agent", DefaultUserAgent)
	} else {
		req.Header.Set("User-Agent", agent)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if resp.Body != nil { // discard reader
			io.Copy(ioutil.Discard, resp.Body)
		}
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(ioutil.Discard, resp.Body) // discard reader
		return nil, errors.New(resp.Status)
	}
	return resp.Body, nil
}

func Accept(url *url.URL, host string, reject, accept []*regexp.Regexp) bool {
	if len(host) == 0 { // single host crawl
		panic("empty crawl host")
	}
	if host != url.Host {
		return false
	}

	name := url.String()
	for i := range reject {
		if reject[i].MatchString(name) {
			return false
		}
	}

	if len(accept) == 0 { // accept all urls
		return true
	}

	for i := range accept {
		if accept[i].MatchString(name) {
			return true
		}
	}
	return false
}

type Robots interface {
	Test(*url.URL) bool
}

// Worker represents a crawler worker implementation.
type Worker struct {
	// GetFunc issues a GET request to the specified URL and returns the
	// response body and an error if any.
	GetFunc func(*url.URL) (io.ReadCloser, error)

	// IsAcceptedFunc can be used to control the crawler.
	IsAcceptedFunc func(*url.URL) bool

	// ProcessFunc can be used to scrape data.
	ProcessFunc func(*html.Node, []byte)

	// Host defines the hostname to crawl. Worker is a single-host crawler.
	Host *url.URL

	// UserAgent defines the user-agent string to use for URL fetching.
	UserAgent string
	Accept    []*regexp.Regexp
	Reject    []*regexp.Regexp

	// Delay to use between requests to a same host if there is not
	// robots.txt crawl delay. The delay starts as soon as the response
	// is received from the host.
	Delay time.Duration

	// MaxEnqueue returns the maximum number of pages visited before
	// stopping the crawl. Note that the Crawler will send its stop signal
	// once this number of visits is reached, but workers may be in the
	// process of visiting other pages, so when the crawling stops, the
	// number of pages visited will be at least MaxEnqueues, possibly more.
	MaxEnqueue int64

	// RobotsAgent defines the user-agent string to use for robots.txt.
	//RobotsAgent string

	Robots Robots
}

func (w *Worker) Get(url *url.URL) (io.ReadCloser, error) {
	if w.GetFunc != nil {
		return w.GetFunc(url)
	}
	return Get(url, w.UserAgent, w.Robots)
}

func (w *Worker) IsAccepted(url *url.URL) bool {
	if w.IsAcceptedFunc != nil {
		return w.IsAcceptedFunc(url)
	}
	return Accept(url, w.Host.Host, w.Reject, w.Accept)
}

func (w *Worker) Process(node *html.Node, data []byte) {
	if w.ProcessFunc != nil {
		w.ProcessFunc(node, data)
	}
}

type worker struct {
	wg     *sync.WaitGroup
	work   chan *url.URL
	done   int
	id     int
	pusher Pusher
	w      *Worker
	logger *log.Logger

	limitReached bool
	closed       bool
}

func (w *worker) printf(format string, args ...interface{}) {
	if w.logger != nil {
		w.logger.Printf(format, args...)
	}
}

func (w *worker) run() {
	for url := range w.work {
		w.printf("worker#%.3d received %q", w.id, url)
		if err := w.fetch(url); err != nil {
			w.printf("worker#%.3d ERROR %q: %v", w.id, url, err)
		}
		w.done++
		if w.w.Delay > 0 {
			time.Sleep(w.w.Delay)
		}
	}
	w.closed = true
	w.wg.Done()
}

func (w *worker) fetch(url *url.URL) error {
	if w.w.Host.Host != url.Host {
		return ErrRejectedURL
	}
	if !url.IsAbs() {
		return ErrNotAbsoluteURL
	}

	body, err := w.w.Get(url)
	if err != nil {
		return err
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	data, _, err = transform.Transform(data, nil)
	if err != nil {
		return err
	}

	node, err := parseHTML(data)
	if err != nil {
		return err
	}

	w.parse(url, node, w.pusher)
	w.w.Process(node, data)
	return nil
}

func (w *worker) parse(parent *url.URL, node *html.Node, pusher Pusher) {
	if w.limitReached || w.closed {
		return
	}

	if node.Type == html.ElementNode && node.Data == "a" {
		for i := range node.Attr {
			if node.Attr[i].Key == "href" && node.Attr[i].Val != "" {
				url, err := normalize(parent, node.Attr[i].Val)
				if err != nil {
					continue
				}

				if !w.w.IsAccepted(url) { // allowed to enqueue
					w.printf("worker#%.3d url parser ERROR %q: rejected url", w.id, url)
					continue
				}
				if err := pusher.Push(url); err != nil {
					switch {
					case err == ErrDuplicateURL:
						// nothing
						w.printf("worker#%.3d url parser ERROR %q: %v", w.id, url, err)

					case err == ErrEmptyURL:
						// nothing
						w.printf("worker#%.3d url parser ERROR %q: %v", w.id, url, err)

					case err == ErrLimitReached:
						w.limitReached = true
						return

					case err == ErrQueueClosed:
						w.closed = true
						return

					default:
						panic("unknown queue error")
					}
				}
			}
		}
	}

	for n := node.FirstChild; n != nil; n = n.NextSibling {
		w.parse(parent, n, pusher)
	}
}

type Crawler struct {
	wg     *sync.WaitGroup
	worker []*worker
	w      *Worker
	i      int // round-robin index
	queue  *Queue
	done   chan struct{}
	logger *log.Logger
}

func New(w *Worker, n uint8, ttl time.Duration, log *log.Logger) *Crawler {
	c := &Crawler{
		queue:  NewQueue(w.MaxEnqueue, ttl),
		worker: make([]*worker, n),
		w:      w,
		wg:     &sync.WaitGroup{},
		done:   make(chan struct{}),
		logger: log,
	}

	for i := uint8(0); i < n; i++ {
		c.worker[i] = &worker{
			work:   make(chan *url.URL), // TODO: buffered channel
			wg:     c.wg,
			id:     int(i) + 1,
			pusher: c.queue,
			w:      w,
			logger: log,
		}
		c.wg.Add(1)
		go c.worker[i].run()
	}

	go c.run()
	return c
}

func (c *Crawler) printf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *Crawler) Start(sm *url.URL, seeds ...*url.URL) error {
	if sm != nil {
		sitemap, err := sitemap.Get(sm.String(), c.w.UserAgent)
		if err != nil {
			return err
		}
		for _, seed := range sitemap.URLSet {
			if err := c.queue.Push(&seed.Loc); err != nil {
				c.printf("enqueue sitemap %q: %v", seed, err)
			}
		}
	}
	for _, seed := range seeds {
		if err := c.queue.Push(seed); err != nil {
			c.printf("enqueue seed %q: %v", seed, err)
		}
	}
	return nil
}

func (c *Crawler) dispatch(url *url.URL) {
	worker := c.worker[c.i]
	worker.work <- url
	c.i++
	if c.i >= len(c.worker) {
		c.i = 0
	}
}

func (c *Crawler) run() {
	for url := range c.queue.Pop() {
		c.dispatch(url)
	}
	for _, w := range c.worker {
		close(w.work)
	}
	c.wg.Wait()

	for _, w := range c.worker {
		c.printf("worker#%.3d closed <done:%d closed:%v>", w.id, w.done, w.closed)
	}

	close(c.done)
}

func (c *Crawler) Done() <-chan struct{} {
	return c.done
}

func (c *Crawler) Close() error {
	return c.queue.Close()
}
