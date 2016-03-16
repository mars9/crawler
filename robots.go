package crawler

import (
	"net/http"
	"net/url"

	"github.com/temoto/robotstxt-go"
)

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

	robots, err := robotstxt.FromResponse(resp)
	if err != nil {
		return fakeAgent{}
	}
	return robots.FindGroup(DefaultUserAgent)
}
