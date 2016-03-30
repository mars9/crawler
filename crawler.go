package crawler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/mars9/crawler/pb"
	"golang.org/x/net/html"
)

const (
	DefaultUserAgent       = "Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0"
	DefaultRobotsUserAgent = "Googlebot (bricktop v1)"
	DefaultTimeToLive      = 3 * DefaultDelay
	DefaultDelay           = 3 * time.Second
)

func Fetch(url *url.URL, agent string, robots Robots) (io.ReadCloser, error) {
	if !url.IsAbs() {
		return nil, errors.New("not an absolute URL")
	}
	if robots != nil && !robots.Test(url.String()) {
		return nil, errors.New("rejected by robots.txt")
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agent)

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
	if !url.IsAbs() || host != url.Host {
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
	Test(string) bool
}

type Crawler struct {
	Accept func(*url.URL, string, []*regexp.Regexp, []*regexp.Regexp) bool
	Fetch  func(*url.URL, string, Robots) (io.ReadCloser, error)

	Host        *url.URL // the hostname to crawl
	RobotsAgent string
	UserAgent   string
	Seeds       []*url.URL
	Accepted    []*regexp.Regexp
	Reject      []*regexp.Regexp
	TTL         time.Duration
	Delay       time.Duration
	MaxVisit    int64

	Robots Robots

	once  sync.Once
	index *index

	mu      sync.Mutex // TODO
	visited int64
	err     error
}

func NewCrawler(config *pb.ConfigRequest) (*Crawler, error) {
	c := &Crawler{}
	if len(config.Host) == 0 {
		return c, errors.New("hostname to crawl not specified")
	}

	host, err := url.Parse(config.Host)
	if err != nil {
		return c, fmt.Errorf("parse hostname: %v", err)
	}
	c.Host = host

	c.Seeds = make([]*url.URL, len(config.Seeds))
	n := 0
	for i := range config.Seeds {
		c.Seeds[n], err = url.Parse(config.Seeds[i])
		if err != nil {
			continue
		}
		n++
	}
	c.Seeds = c.Seeds[:n]

	c.Accepted = make([]*regexp.Regexp, len(config.Accept))
	c.Reject = make([]*regexp.Regexp, len(config.Reject))
	for i := range config.Accept {
		c.Accepted[i], err = regexp.Compile(config.Accept[i])
		if err != nil {
			return c, fmt.Errorf("compile accept#%d: %v", err)
		}
	}
	for i := range config.Reject {
		c.Reject[i], err = regexp.Compile(config.Reject[i])
		if err != nil {
			return c, fmt.Errorf("compile reject#%d: %v", err)
		}
	}

	c.TTL = time.Duration(config.TimeToLive)
	if c.TTL == 0 {
		c.TTL = DefaultTimeToLive
	}

	c.RobotsAgent = config.RobotsAgent
	c.UserAgent = config.UserAgent
	if c.RobotsAgent == "" {
		c.RobotsAgent = DefaultRobotsUserAgent
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}

	c.Delay = time.Duration(config.Delay)
	c.MaxVisit = config.MaxVisit

	return c, nil
}

func (c *Crawler) init() {
	c.once.Do(func() {
		if c.Fetch == nil {
			c.Fetch = Fetch
		}
		if c.Accept == nil {
			c.Accept = Accept
		}
		c.index = newIndex()
	})
}

func (c *Crawler) Do(url *url.URL, producer *Producer) {
	c.init()
	if c.err != nil {
		producer.Close()
		return
	}
	if c.err = c.fetch(url, producer); c.err != nil {
		return
	}
	//	log.Printf("fetched: %s", url)

	c.mu.Lock()
	if c.MaxVisit > 0 && c.visited >= c.MaxVisit {
		log.Println("-----------------------------")
		producer.Close()
		c.err = errors.New("crawler producer closed")
	}
	log.Println(c.visited, c.MaxVisit)
	c.visited++
	c.mu.Unlock()
}

func (c *Crawler) fetch(url *url.URL, producer *Producer) error {
	body, err := c.Fetch(url, c.UserAgent, c.Robots)
	if err != nil {
		return err
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}

	// TODO: err signals closed Producer
	if err = c.parse(node, producer); err != nil {
		return err
	}
	return err
}

func (c *Crawler) parse(node *html.Node, prod *Producer) error {
	if node.Type == html.ElementNode && node.Data == "a" {
		for i := range node.Attr {
			if node.Attr[i].Key == "href" && node.Attr[i].Val != "" {
				url, err := normalize(c.Host, node.Attr[i].Val)
				if err != nil {
					continue
				}

				if !c.Accept(url, c.Host.Host, c.Reject, c.Accepted) {
					continue
				}
				if c.index.Has(url) { // already visited
					continue
				}

				if err := prod.Send(url); err != nil {
					return err
				}
			}
		}
	}

	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if err := c.parse(n, prod); err != nil {
			return err
		}
	}
	return nil
}
