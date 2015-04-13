package crawler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mars9/crawler/robotstxt"
)

// ParseFunc implements Crawler Parse.
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

	client := &http.Client{}
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

// Config defines the default crawler configuration.
type Config struct {
	Domain      string        `json:"domain"`
	UserAgent   string        `json:"user_agent"`
	RobotsAgent string        `json:"robots_agent""`
	Seeds       []string      `json:"seeds"`
	Accept      []string      `json:"accept"`
	Reject      []string      `json:"reject"`
	MaxVisit    uint32        `json:"max_visit"`
	TimeToLive  time.Duration `json:"time_to_live"`
	Delay       time.Duration `json:"delay"`
}

// New returns a default Crawler implementation.
func New(config Config, fn ParseFunc) (Crawler, error) {
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
	if config.TimeToLive == 0 {
		config.TimeToLive = DefaultTimeToLive
	}
	if config.UserAgent == "" {
		config.UserAgent = DefaultUserAgent
	}
	if config.RobotsAgent == "" {
		config.RobotsAgent = DefaultRobotsUserAgent
	}

	return &defCrawler{
		domain:    domain,
		userAgent: config.UserAgent,
		seeds:     seeds,
		accept:    accept,
		reject:    reject,
		ttl:       config.TimeToLive,
		delay:     config.Delay,
		maxVisit:  config.MaxVisit,
		parseFunc: fn,
		client:    &http.Client{Transport: &http.Transport{}},
		agent:     fetchUserAgent(domain, config.RobotsAgent),
		visited:   make(map[string]bool),
	}, nil
}

type defCrawler struct {
	domain    *url.URL
	userAgent string
	seeds     []*url.URL
	accept    []*regexp.Regexp
	reject    []*regexp.Regexp
	ttl       time.Duration
	delay     time.Duration
	maxVisit  uint32
	parseFunc ParseFunc
	client    *http.Client
	agent     userAgent
	mu        sync.Mutex
	visited   map[string]bool
}

var errorReg = regexp.MustCompile(`[0-9]* `)

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

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.ToLower(errorReg.ReplaceAllString(resp.Status, ""))
		return nil, errors.New(msg)
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

func (c *defCrawler) MaxVisit() uint32     { return c.maxVisit }
func (c *defCrawler) Seeds() []*url.URL    { return c.seeds }
func (c *defCrawler) Domain() *url.URL     { return c.domain }
func (c *defCrawler) Delay() time.Duration { return c.delay }
func (c *defCrawler) TTL() time.Duration   { return c.ttl }
