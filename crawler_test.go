package crawler

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestWorker() *Worker {
	host, _ := url.Parse("http://example.com")

	return &Worker{
		GetFunc: func(url *url.URL) (io.ReadCloser, error) {
			data := "<html><head></head><body></body></html>"
			return ioutil.NopCloser(strings.NewReader(data)), nil
		},
		Host: host,
	}
}

func TestWorkerClose(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}
	want := 10
	w := &worker{
		wg:   wg,
		work: make(chan *url.URL),
		w:    newTestWorker(),
	}

	wg.Add(1)
	go w.run()

	for i := 0; i < want; i++ {
		u, _ := url.Parse(fmt.Sprintf("http://example.com/site%d", i))
		w.work <- u
	}
	close(w.work)

	wg.Wait()

	if !w.closed {
		t.Fatalf("worker: expected closed worker")
	}
	if w.done != want {
		t.Fatalf("worker: expected %d crawled url, got %d", want, w.done)
	}
}

func TestCrawlerClose(t *testing.T) {
	t.Parallel()

	c := New(newTestWorker(), 8, time.Millisecond*2, nil)
	time.Sleep(time.Millisecond * 5)
	<-c.Done()

	for _, w := range c.worker {
		if !w.closed {
			t.Fatalf("crawler: expected closed worker")
		}
	}

	u, _ := url.Parse("http://example.com")
	if err := c.queue.Push(u); err != ErrQueueClosed {
		t.Fatalf("crawler: expected ErrQueueClosed, got %v", err)
	}
}

func TestAcceptFunc(t *testing.T) {
	reject := []*regexp.Regexp{
		regexp.MustCompile("http://example.com/index.html"),
		regexp.MustCompile("http://example.com/notwant*"),
	}
	accept := []*regexp.Regexp{
		regexp.MustCompile(`http://example.com/site1\.html`),
		regexp.MustCompile(`http://example.com/site2\.html`),
		regexp.MustCompile("http://example.com/index1*"),
	}

	// test reject
	for _, urlStr := range []string{
		"http://google.com",
		"http://example.com/index.html",
		"http://example.com/notwant",
		"http://example.com/notwant1",
		"http://example.com/notwant2",

		"http://example.com/site3.html",
	} {
		u, _ := url.Parse(urlStr)
		if Accept(u, "example.com", reject, accept) {
			t.Fatalf("accept %q: expected false, got true", urlStr)
		}
	}

	// test accept all
	for _, urlStr := range []string{
		"http://example.com/site1.html",
		"http://example.com/xnotwan",
		"http://example.com/index1.html",
	} {
		u, _ := url.Parse(urlStr)
		if !Accept(u, "example.com", reject, nil) {
			t.Fatalf("accept %q: expected true, got false", urlStr)
		}
	}

	// test accept
	for _, urlStr := range []string{
		"http://example.com/site1.html",
		"http://example.com/site2.html",
		"http://example.com/index1.html",
	} {
		u, _ := url.Parse(urlStr)
		if !Accept(u, "example.com", reject, accept) {
			t.Fatalf("accept %q: expected true, got false", urlStr)
		}
	}
}

func startGetServer(t *testing.T) *httptest.Server {
	h := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/index" {
			w.Write([]byte("hello world"))
			return
		} else if req.URL.Path == "/404" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestGetFunc(t *testing.T) {
	t.Parallel()

	s := startGetServer(t)
	defer s.Close()

	u, _ := url.Parse(s.URL)

	u.Path = "/index"
	body, err := Get(u, "agent", nil)
	if err != nil {
		t.Fatalf("get: expected <nil> error, got %v", err)
	}
	data, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatalf("get: read body: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("get: expected \"hello world\", got %q", data)
	}

	u.Path = "/404"
	if body, err = Get(u, "agent", nil); err == nil {
		t.Fatalf("get: expected 404 Not Found, got <nil>")
	}
	if err.Error() != "404 Not Found" {
		t.Fatalf("get: expected 404 Not Found, got %v", err)
	}
	if body != nil {
		t.Fatalf("get: expected <nil> body")
	}

	u.Path = "xxx"
	if _, err = Get(u, "agent", nil); err == nil {
		t.Fatalf("get: expected 400 Bad Request, got <nil>")
	}
	if err.Error() != "400 Bad Request" {
		t.Fatalf("get: expected 400 Bad Request, got %v", err)
	}
	if body != nil {
		t.Fatalf("get: expected <nil> body")
	}
}
