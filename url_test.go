package crawler

import (
	"net/url"
	"testing"
)

func testURLNormalize1(t *testing.T) {
	domain, _ := url.Parse("http://google.com")
	want := []string{
		"http://google.com/search?q=golang",
		"http://google.com/search",
		"http://google.com",
		"http://google.com/search",
		"http://google.com/search",
		"http://google.com/search#fragment",
		"http://google.com/search?q=golang",
		"http://google.com/search?q=golang",
		"http://google.com#fragment",
		"http://google.com/search",
	}

	for i, link := range []string{
		"http://google.com/search?q=golang",
		"http://google.com/search",
		"http://google.com",
		"/search",
		"search",
		"/search#fragment",
		"/search?q=golang",
		"search?q=golang",
		"#fragment",
		"//google.com/search",
	} {
		url, err := normalize(domain, link)
		if err != nil {
			t.Fatalf("normalize url %q: %v", link, err)
		}
		if want[i] != url.String() {
			t.Fatalf("normalize url: expected %q, got %q", want[i], url)
		}
	}
}

func testURLNormalize2(t *testing.T) {
	domain, _ := url.Parse("http://google.com/sub")
	want := []string{
		"http://google.com/sub/search?q=golang",
		"http://google.com/sub/search",
		"http://google.com/sub",
		"http://google.com/search",
		"http://google.com/sub/search",
		"http://google.com/search#fragment",
		"http://google.com/search?q=golang",
		"http://google.com/sub/search?q=golang",
		"http://google.com/sub#fragment",
		"http://google.com/sub/search",
	}

	for i, link := range []string{
		"http://google.com/sub/search?q=golang",
		"http://google.com/sub/search",
		"http://google.com/sub",
		"/search",
		"search",
		"/search#fragment",
		"/search?q=golang",
		"search?q=golang",
		"#fragment",
		"//google.com/sub/search",
	} {
		url, err := normalize(domain, link)
		if err != nil {
			t.Fatalf("normalize url %q: %v", link, err)
		}
		if want[i] != url.String() {
			t.Fatalf("normalize url: expected %q, got %q", want[i], url)
		}
	}
}

func TestURLNormalize(t *testing.T) {
	testURLNormalize1(t)
	testURLNormalize2(t)
}
