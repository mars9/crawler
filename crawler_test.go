package crawler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"
)

func startTestServer(t *testing.T) *httptest.Server {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current working directory: %v", err)
	}
	testDir := http.Dir(path.Join(cwd, "testdata"))

	s := httptest.NewServer(http.FileServer(testDir))
	return s
}

func TestBasicCrawler(t *testing.T) {
	t.Parallel()

	s := startTestServer(t)
	defer s.Close()

	w := &Worker{}
	w.Host, _ = url.Parse(s.URL)

	c := New(w, 8, time.Millisecond*20)
	c.Start(nil, w.Host)

	<-c.Done()
}
