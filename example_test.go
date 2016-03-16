package crawler

import (
	"log"
	"net/url"
	"time"

	"github.com/mars9/crawler/pb"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
)

func parse(url *url.URL, root *html.Node, body []byte) error {
	log.Printf("FOUND: %q\n", url)
	return nil
}

func ExampleStart() {
	config := &pb.Config{
		Domain: "https://golang.org",
		Seeds: []string{
			"https://golang.org/doc/",
			"https://golang.org/pkg/",
		},
		TimeToLive: int64(time.Second * 1),
		Delay:      int64(time.Second * 1),
	}

	c, err := New(config, parse)
	if err != nil {
		log.Fatalf("create crawler: %v", err)
	}
	Start(context.TODO(), c)
}
