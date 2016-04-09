package rpc

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"time"

	crawler "github.com/mars9/crawler"
	"github.com/mars9/crawler/proto"
)

func newRequest(req *proto.StartRequest) (*crawler.Worker, error) {
	w := &crawler.Worker{}
	if len(req.Hostname) == 0 {
		return nil, errors.New("hostname to crawl not specified")
	}
	host, err := url.Parse(req.Hostname)
	if err != nil {
		return nil, fmt.Errorf("parse hostname: %v", err)
	}
	w.Host = host

	w.Accept = make([]*regexp.Regexp, len(req.Accept))
	w.Reject = make([]*regexp.Regexp, len(req.Reject))
	for i := range req.Accept {
		w.Accept[i], err = regexp.Compile(req.Accept[i])
		if err != nil {
			return nil, fmt.Errorf("compile accept#%d: %v", err)
		}
	}

	for i := range req.Reject {
		w.Reject[i], err = regexp.Compile(req.Reject[i])
		if err != nil {
			return nil, fmt.Errorf("compile reject#%d: %v", err)
		}
	}

	w.RobotsAgent = req.RobotsAgent
	w.UserAgent = req.UserAgent
	if w.RobotsAgent == "" {
		w.RobotsAgent = crawler.DefaultRobotsAgent
	}
	if w.UserAgent == "" {
		w.UserAgent = crawler.DefaultUserAgent
	}

	w.Delay = time.Duration(req.Delay)
	w.MaxEnqueue = req.MaxEnqueue
	return w, nil
}

func parseSeeds(seeds []string) []*url.URL {
	s := make([]*url.URL, 0, len(seeds))
	for _, seed := range seeds {
		u, err := url.Parse(seed)
		if err != nil {
			continue
		}
		if !u.IsAbs() {
			continue
		}
		s = append(s, u)
	}
	return s
}

func verifyWorkers(workers int32) uint8 {
	switch {
	case workers > math.MaxUint8:
		return math.MaxUint8
	case workers <= 0:
		return 1
	default:
		return uint8(workers)
	}
}
