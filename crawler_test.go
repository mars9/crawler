package crawler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

type walker int

func (w *walker) walk(path string, info os.FileInfo, err error) error {
	*w++
	return nil
}

func startBasicTestServer(t *testing.T) (*httptest.Server, int) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current working directory: %v", err)
	}
	testPath := path.Join(cwd, "testdata", "basic")
	testDir := http.Dir(testPath)

	w := new(walker)
	if err = filepath.Walk(testPath, w.walk); err != nil {
		t.Fatalf("walk test directory %q: %v", testPath, err)
	}

	s := httptest.NewServer(http.FileServer(testDir))
	return s, int(*w)
}

func TestBasicCrawler(t *testing.T) {
	t.Parallel()

	s, want := startBasicTestServer(t)
	defer s.Close()

	w := &Worker{}
	w.Host, _ = url.Parse(s.URL)

	c := New(w, 8, time.Millisecond*20)
	c.Start(nil, w.Host)

	<-c.Done()

	got := 0
	for _, w := range c.worker {
		got += w.done
	}
	if got != want {
		t.Fatalf("basic crawler: expected %d hits, got %d", want, got)
	}
}
