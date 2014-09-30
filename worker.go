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

	"code.google.com/p/go.net/html"
	"code.google.com/p/goprotobuf/proto"
	pb "github.com/mars9/crawler/crawlerpb"
	"github.com/mars9/crawler/robotstxt"
)

// ParseFunc implements Crawler Parse.
type ParseFunc func(url *url.URL, node *html.Node) (err error)

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

// New returns a default Crawler implementation.
func New(args *pb.Crawler, fn ParseFunc) (Crawler, error) {
	domain, err := url.Parse(args.GetDomain())
	if err != nil {
		return nil, err
	}

	seeds, n := make([]*url.URL, len(args.Seeds)), 0
	for i := range args.Seeds {
		seeds[n], err = url.Parse(args.Seeds[i])
		if err != nil {
			continue
		}
		n++
	}
	seeds = seeds[:n]

	accept := make([]*regexp.Regexp, len(args.Accept))
	reject := make([]*regexp.Regexp, len(args.Reject))
	for i := range args.Accept {
		accept[i], err = regexp.Compile(args.Accept[i])
		if err != nil {
			return nil, err
		}
	}
	for i := range args.Reject {
		reject[i], err = regexp.Compile(args.Reject[i])
		if err != nil {
			return nil, err
		}
	}
	if args.GetTimeToLive() == 0 {
		args.TimeToLive = proto.Int64(int64(DefaultTimeToLive))
	}
	if args.GetUserAgent() == "" {
		args.UserAgent = proto.String(DefaultUserAgent)
	}
	if args.GetRobotsAgent() == "" {
		args.RobotsAgent = proto.String(DefaultRobotsUserAgent)
	}

	return &defCrawler{
		domain:    domain,
		userAgent: args.GetUserAgent(),
		seeds:     seeds,
		accept:    accept,
		reject:    reject,
		ttl:       time.Duration(args.GetTimeToLive()),
		delay:     time.Duration(args.GetDelay()),
		maxVisit:  args.GetMaxVisit(),
		parseFunc: fn,
		client:    &http.Client{Transport: &http.Transport{}},
		agent:     fetchUserAgent(domain, args.GetRobotsAgent()),
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

func (c *defCrawler) Parse(url *url.URL, node *html.Node) error {
	return c.parseFunc(url, node)
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
