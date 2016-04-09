package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	crawler "github.com/mars9/crawler"
	"github.com/mars9/crawler/proto"
)

func NewRequest(req *proto.CrawlerRequest) (*crawler.Worker, error) {
	w := &crawler.Worker{}
	if len(req.Host) == 0 {
		return nil, errors.New("hostname to crawl not specified")
	}
	host, err := url.Parse(req.Host)
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
