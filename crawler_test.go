package crawler

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"golang.org/x/net/context"
)

var (
	pageData = `<html><meta><title>page %d</title></meta>
<body>
	<a href="%s/page%.4d">page%.4d</a>
	<a href="%s/page%.4d">page%.4d</a>

	<a href="http://example.com">example.com</a>
	<a href="http://golang.org">golang</a>
	<a href="http://haskell.org">haskell</a>
</body>
</html>
`
	indexData = `<a href="%s/%s">%s</a>
`
)

func initDirectory(t *testing.T, testServerAddr, testDataDir string) {
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", testDataDir, err)
	}

	type page struct {
		Path string
		Body []byte
	}
	for i := 0; i < 20; i++ {
		page := page{
			Path: path.Join(testDataDir, fmt.Sprintf("page%.4d", i)),
			Body: []byte(fmt.Sprintf(pageData, i, testServerAddr, i+20, i+20)),
		}
		if err := ioutil.WriteFile(page.Path, page.Body, 0644); err != nil {
			t.Fatalf("write %s: %v", page.Path, err)
		}
	}
	for i := 20; i < 40; i++ {
		page := page{
			Path: path.Join(testDataDir, fmt.Sprintf("page%.4d", i)),
			Body: []byte(fmt.Sprintf(pageData, i, testServerAddr, i-20, i-20)),
		}
		if err := ioutil.WriteFile(page.Path, page.Body, 0644); err != nil {
			t.Fatalf("write %s: %v", page.Path, err)
		}
	}

	var indices []page
	n := 0
	for i := 0; i < 4; i++ {
		fname := fmt.Sprintf("index%.4d", i)
		page := page{
			Path: path.Join(testDataDir, fname),
			//Body: []byte(fmt.Sprintf(pageData, i)),
		}
		for j := 0; j < 5; j++ {
			pname := fmt.Sprintf("page%.4d", n+j)
			body := []byte(fmt.Sprintf(indexData, testServerAddr, pname, pname))
			page.Body = append(page.Body, body...)
		}
		n += 5

		indices = append(indices, page)
		if err := ioutil.WriteFile(page.Path, page.Body, 0644); err != nil {
			t.Fatalf("write %s: %v", page.Path, err)
		}
	}
}

func generateUUID() string {
	now := uint32(time.Now().Unix())
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%04x%08x",
		now, b[0:2], b[2:4], b[4:6], b[6:8], b[8:])
}

func TestCrawlerIntegration(t *testing.T) {
	t.Parallel()

	testDataDir := fmt.Sprintf("/tmp/test-crawler-data-%s", generateUUID())
	defer os.RemoveAll(testDataDir)
	server := httptest.NewServer(http.FileServer(http.Dir(testDataDir)))
	defer server.Close()

	initDirectory(t, server.URL, testDataDir)
	config := Config{
		Domain: server.URL,
		Seeds: []string{
			server.URL + "/index0000",
			server.URL + "/index0001",
			server.URL + "/index0002",
			server.URL + "/index0003",
		},
		Accept:     []string{server.URL},
		Reject:     []string{},
		TimeToLive: time.Millisecond * 50,
		Delay:      0,
	}

	want := make(map[string]bool)
	for i := 0; i < 4; i++ {
		want[fmt.Sprintf("%s/index%.4d", server.URL, i)] = true
	}
	for i := 0; i < 40; i++ {
		want[fmt.Sprintf("%s/page%.4d", server.URL, i)] = true
	}

	got := make(map[string]bool)
	c, err := New(config, func(url *url.URL, body []byte) error {
		got[url.String()] = true
		return nil
	})
	if err != nil {
		t.Fatal("default crawler: %v", err)
	}

	Start(context.Background(), c, 5)

	assert(t, "urls", want, got)
}

func TestCrawlerMaxVisit(t *testing.T) {
	t.Parallel()

	testDataDir := fmt.Sprintf("/tmp/test-crawler-data-%s", generateUUID())
	defer os.RemoveAll(testDataDir)
	server := httptest.NewServer(http.FileServer(http.Dir(testDataDir)))
	defer server.Close()

	initDirectory(t, server.URL, testDataDir)
	config := Config{
		Domain: server.URL,
		Seeds: []string{
			server.URL + "/index0000",
			server.URL + "/index0001",
			server.URL + "/index0002",
			server.URL + "/index0003",
		},
		Accept:     []string{server.URL},
		MaxVisit:   3,
		Reject:     []string{},
		TimeToLive: time.Millisecond * 50,
		Delay:      0,
	}

	want := make(map[string]bool)
	for i := 0; i < 3; i++ {
		want[fmt.Sprintf("%s/index%.4d", server.URL, i)] = true
	}

	got := make(map[string]bool)
	c, err := New(config, func(url *url.URL, body []byte) error {
		got[url.String()] = true
		return nil
	})
	if err != nil {
		t.Fatal("default crawler: %v", err)
	}

	Start(context.Background(), c, 5)

	assert(t, "urls", want, got)
}
