// Package crawler provides a crawler implementation.
package crawler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	pb "github.com/mars9/crawler/crawlerpb"
	"github.com/mars9/crawler/robotstxt"
)

// Default options.
const (
	DefaultUserAgent       = "Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0"
	DefaultRobotsUserAgent = "Googlebot (crawlbot v1)"
	DefaultTimeToLive      = 3 * DefaultCrawlDelay
	DefaultCrawlDelay      = 3 * time.Second
)

// Crawler represents a crawler implementation.
type Crawler interface {
	// Fetch issues a GET to the specified URL and returns the response body
	// and an error if any.
	Fetch(url *url.URL) (rc io.ReadCloser, err error)

	// Parse is called when visiting a page. Parse receives a http response
	// body and should return an error, if any.
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
	MaxVisit() (max int64)

	// Delay returns the time to wait between each request to the same host.
	// The delay starts as soon as the response is received from the host.
	Delay() (delay time.Duration)

	// TTL returns the duration that a crawler goroutine can wait without
	// receiving new commands to fetch. If the idle time-to-live is reached,
	// the crawler goroutine is stopped and its resources are released. This
	// can be especially useful for long-running crawlers.
	TTL() (timeout time.Duration)
}

// ParseFunc implements Crawler.Parse.
type ParseFunc func(url *url.URL, body []byte) (err error)

type userAgent interface {
	Test(path string) (ok bool)
}

type fakeAgent struct{}

func (f fakeAgent) Test(path string) bool { return true }

func fetchUserAgent(domain *url.URL, robotsAgent string) userAgent {
	req, err := http.NewRequest("GET", domain.String()+"/robots.txt", nil)
	if err != nil {
		return fakeAgent{}
	}
	req.Header.Set("User-Agent", robotsAgent)

	client := &http.Client{} // TODO: reuse client / client pool
	resp, err := client.Do(req)
	if err != nil {
		return fakeAgent{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fakeAgent{}
	}

	robots, err := robotstxt.Parse(resp.Body)
	if err != nil {
		return fakeAgent{}
	}
	return robots.FindGroup(DefaultUserAgent)
}

type defCrawler struct {
	domain    *url.URL
	userAgent string
	seeds     []*url.URL
	accept    []*regexp.Regexp
	reject    []*regexp.Regexp
	ttl       time.Duration
	delay     time.Duration
	maxVisit  int64
	parseFunc ParseFunc
	agent     userAgent

	mu      sync.Mutex
	visited map[string]bool
}

// New returns a default Crawler implementation.
func New(config *pb.Config, parse ParseFunc) (Crawler, error) {
	domain, err := url.Parse(config.Domain)
	if err != nil {
		return nil, err
	}

	seeds, n := make([]*url.URL, len(config.Seeds)), 0
	for i := range config.Seeds {
		seeds[n], err = url.Parse(config.Seeds[i])
		if err != nil {
			continue
		}
		n++
	}
	seeds = seeds[:n]

	accept := make([]*regexp.Regexp, len(config.Accept))
	reject := make([]*regexp.Regexp, len(config.Reject))
	for i := range config.Accept {
		accept[i], err = regexp.Compile(config.Accept[i])
		if err != nil {
			return nil, err
		}
	}
	for i := range config.Reject {
		reject[i], err = regexp.Compile(config.Reject[i])
		if err != nil {
			return nil, err
		}
	}

	timeToLive := time.Duration(config.TimeToLive)
	if timeToLive == 0 {
		timeToLive = DefaultTimeToLive
	}

	if config.UserAgent == "" {
		config.UserAgent = DefaultUserAgent
	}
	if config.RobotsAgent == "" {
		config.RobotsAgent = DefaultRobotsUserAgent
	}

	agent := fetchUserAgent(domain, config.RobotsAgent)
	return &defCrawler{
		domain:    domain,
		userAgent: config.UserAgent,
		seeds:     seeds,
		accept:    accept,
		reject:    reject,
		ttl:       timeToLive,
		delay:     time.Duration(config.Delay),
		maxVisit:  config.MaxVisit,
		parseFunc: parse,
		agent:     agent,
		visited:   make(map[string]bool),
	}, nil
}

func (c *defCrawler) Fetch(url *url.URL) (io.ReadCloser, error) {
	c.mu.Lock()
	if _, found := c.visited[url.String()]; found {
		c.mu.Unlock()
		return nil, errors.New("already visited")
	}
	c.visited[url.String()] = true
	c.mu.Unlock()

	if !c.agent.Test(url.String()) {
		return nil, errors.New("rejected by robots.txt")
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	// TODO: reuse client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *defCrawler) Parse(url *url.URL, body []byte) error {
	return c.parseFunc(url, body)
}

func (c *defCrawler) Accept(url *url.URL) bool {
	for i := range c.reject {
		if c.reject[i].MatchString(url.String()) {
			return false
		}
	}
	for i := range c.accept {
		if c.accept[i].MatchString(url.String()) {
			return true
		}
	}
	return false
}

func (c *defCrawler) MaxVisit() int64      { return c.maxVisit }
func (c *defCrawler) Seeds() []*url.URL    { return c.seeds }
func (c *defCrawler) Domain() *url.URL     { return c.domain }
func (c *defCrawler) Delay() time.Duration { return c.delay }
func (c *defCrawler) TTL() time.Duration   { return c.ttl }
