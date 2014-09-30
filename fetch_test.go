package crawler

import (
	"bytes"
	"io"
	"net/url"
	"testing"
	"time"

	"code.google.com/p/go.net/html"
)

type parseTestCrawler struct {
	domain *url.URL
	ttl    time.Duration
}

func (p parseTestCrawler) Fetch(url *url.URL) (io.ReadCloser, error) { return nil, nil }
func (p parseTestCrawler) Parse(url *url.URL, body []byte) error     { return nil }
func (p parseTestCrawler) Seeds() []*url.URL                         { return nil }
func (p parseTestCrawler) Domain() *url.URL                          { return p.domain }
func (p parseTestCrawler) MaxVisit() uint32                          { return 0 }
func (p parseTestCrawler) Accept(url *url.URL) bool                  { return true }
func (p parseTestCrawler) Delay() time.Duration                      { return 0 }
func (p parseTestCrawler) TTL() time.Duration                        { return p.ttl }

func TestParseHTML(t *testing.T) {
	t.Parallel()

	domain, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c := parseTestCrawler{domain: domain}

	buf := bytes.NewBuffer(parseTestHTMLData)
	pushc := make(chan *url.URL)
	go func() {
		defer close(pushc)

		node, err := html.Parse(buf)
		if err != nil {
			t.Fatal(err)
		}

		if err := parseHTML(node, c, pushc); err != nil {
			t.Fatalf("parsing html test data: %v", err)
		}
	}()

	var got []string
	for url := range pushc {
		got = append(got, url.String())
	}
	for i := range expectedParseResult {
		assert(t, "url", expectedParseResult[i], got[i])
	}

}

var parseTestHTMLData = []byte(`
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="de" lang="de">
<head>
	<title>file1</title>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
	<meta name="description" content="the description" />
	<meta name="keywords" content="keyword1, keyword2, keyword3" />
	<meta name="robots" content="INDEX,FOLLOW" />
	<meta name="author" content="the author">

	<link rel="stylesheet" type="text/css" href="http://example.com/style1.css" media="all" />
	<link rel="stylesheet" type="text/css" href="http://example.com/style2.css" media="all" />
	<link rel="stylesheet" type="text/css" href="http://example.com/style3.css" media="all" />

	<script type="text/javascript" src="http://example.com/script1.js"></script>
	<script type="text/javascript" src="http://example.com/script2.js"></script>
	<script type="text/javascript" src="http://example.com/script3.js"></script>
	<script type="text/javascript">
	//<![CDATA[
		optionalZipCountries = [];
	//]]>
	</script>
	<script type="text/javascript">
		function testFunc() {}
	</script>
</head>
<body>
	<a href="http://example.com/site1.html">site1</a>
	<a href="http://example.com/site2.html">site2</a>
	<a href="http://example.com/site3.html">site3</a>
	<a href="http://example.com/site4.html">site4</a>
	<a href="http://example.com/site5.html">site5</a>
	<a href="http://example.com/site6.html">site6</a>
	<a href="http://example.com/site6.html">site6</a>

	<a href="/site7.html">site7</a>
	<a href="/site8.html">site7</a>

	<a href="#">#</a>
	<a href="#goto">goto</a>

	<a href="http://golang.org">go</a>
	<a href="http://golang.org">go</a>
	<a href="http://google.com/?q=golang">search</a>
</body>
</html>
`)

var expectedParseResult = []string{
	"http://example.com/site1.html",
	"http://example.com/site2.html",
	"http://example.com/site3.html",
	"http://example.com/site4.html",
	"http://example.com/site5.html",
	"http://example.com/site6.html",
	"http://example.com/site6.html",
	"http://example.com/site7.html",
	"http://example.com/site8.html",
	"http://example.com/",
	"http://example.com/#goto",
	"http://golang.org",
	"http://golang.org",
	"http://google.com/?q=golang",
}
