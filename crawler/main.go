package main

import (
	"flag"
	"log"
	"net/url"
	"time"

	crawler "github.com/mars9/crawler"
	"github.com/mars9/crawler/proto"
)

var (
	ttl    = flag.Duration("ttl", time.Second*60, "duration queue can wait without receiving a URL")
	worker = flag.Uint("worker", 8, "concurrent crawl worker")
)

func main() {
	flag.Parse()

	req := &proto.CrawlerRequest{
		Host: "https://www.allesbecher.at",
		Seeds: []string{
			"https://www.allesbecher.at/einwegbecher/plastikbecher/",
			"https://www.allesbecher.at/mehrwegbecher/",
		},
		Reject: []string{
			"https://www.allesbecher.at/catalog/product_compare*",
			"https://www.allesbecher.at/info/becher-bedrucken*",
			"https://www.allesbecher.at/info*",
			"https://www.allesbecher.at/contacts*",
			"https://www.allesbecher.at/customer*",
			"https://www.allesbecher.at/checkout*",
		},
		Delay:      int64(time.Second * 3),
		MaxEnqueue: 100,
	}
	seeds := make([]*url.URL, 0, len(req.Seeds))
	for _, seed := range req.Seeds {
		u, err := url.Parse(seed)
		if err != nil {
			continue
		}
		seeds = append(seeds, u)
	}

	w, err := NewRequest(req)
	if err != nil {
		log.Fatalf("initialize crawler worker: %v", err)
	}

	c := crawler.New(w, uint8(*worker), *ttl)
	if err = c.Start(nil, seeds...); err != nil {
		log.Fatalf("start crawler: %v", err)
	}
	<-c.Done()
}
