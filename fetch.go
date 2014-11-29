package crawler

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func normalize(domain *url.URL, link string) (*url.URL, error) {
	link = strings.TrimSpace(link)
	switch {
	case strings.HasPrefix(link, "//"):
		link = domain.Scheme + ":" + link
	case strings.HasPrefix(link, "/"):
		link = domain.Scheme + "://" + domain.Host + link
	case strings.HasPrefix(link, "#"):
		link = domain.Scheme + "://" + domain.Host + "/" + link
	}

	return url.Parse(link) // verify, normalize url
}

func parseHTML(node *html.Node, c Crawler, push chan<- *url.URL) error {
	var err error
	if node.Type == html.ElementNode && node.Data == "a" {
		for i := range node.Attr {
			if node.Attr[i].Key == "href" && node.Attr[i].Val != "" {
				url, err := normalize(c.Domain(), node.Attr[i].Val)
				if err != nil {
					continue
				}
				if !c.Accept(url) {
					continue
				}
				push <- url
			}
		}
	}

	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if err = parseHTML(n, c, push); err != nil {
			return err
		}
	}
	return nil
}

// Fetch issues a GET to the specified URL. Fetch follows redirects up to
// a maximum of 10 redirects. Fetch sends all found links to push and
// afterwards calls Crawler Parse, if not nil.
func Fetch(url *url.URL, c Crawler, push chan<- *url.URL) error {
	body, err := c.Fetch(url)
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

	//prefix, err := regexp.Compile(`^` + c.Domain().String())
	//if err != nil {
	//	return err
	//}
	if err = parseHTML(node, c, push); err != nil {
		return err
	}
	if c.Parse != nil {
		return c.Parse(url, data)
	}
	return err
}
