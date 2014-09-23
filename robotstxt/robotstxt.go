// Package robotstxt implements the robots.txt Exclusion Protocol as
// specified in
// http://www.robotstxt.org/wc/robots.html
package robotstxt

// Comments explaining the logic are taken from either the Google's spec:
// https://developers.google.com/webmasters/control-crawl-index/docs/robots_txt

import (
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"time"
)

type Robots struct {
	groups      []*Group
	allowAll    bool
	disallowAll bool
	Sitemaps    []string
}

type Group struct {
	agents     []string
	rules      []*rule
	CrawlDelay time.Duration
}

type rule struct {
	path    string
	allow   bool
	pattern *regexp.Regexp
}

type ParseError struct {
	Errs []error
}

func newParseError(errs []error) *ParseError {
	return &ParseError{errs}
}

func (e ParseError) Error() string {
	var b bytes.Buffer

	b.WriteString("Parse error(s): " + "\n")
	for _, er := range e.Errs {
		b.WriteString(er.Error() + "\n")
	}
	return b.String()
}

var allowAll = &Robots{allowAll: true}
var disallowAll = &Robots{disallowAll: true}
var emptyGroup = &Group{}

func Parse(r io.Reader) (*Robots, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var errs []error

	// special case (probably not worth optimization?)
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return allowAll, nil
	}

	sc := newByteScanner("bytes", true)
	//sc.Quiet = !print_errors
	sc.Feed(body, true)
	var tokens []string
	tokens, err = sc.ScanAll()
	if err != nil {
		return nil, err
	}

	// special case worth optimization
	if len(tokens) == 0 {
		return allowAll, nil
	}

	rd := &Robots{}
	parser := newParser(tokens)
	rd.groups, rd.Sitemaps, errs = parser.parseAll()
	if len(errs) > 0 {
		return nil, newParseError(errs)
	}

	return rd, nil
}

func (r *Robots) TestAgent(path, agent string) bool {
	if r.allowAll {
		return true
	}
	if r.disallowAll {
		return false
	}

	// Find a group of rules that applies to this agent.
	// From Google's spec:
	// The user-agent is non-case-sensitive.
	if g := r.FindGroup(agent); g != nil {
		// Find a rule that applies to this url
		if r := g.findRule(path); r != nil {
			return r.allow
		}
	}

	// From Google's spec:
	// By default, there are no restrictions for crawling for the
	// designated crawlers.
	return true
}

// From Google's spec:
// Only one group of group-member records is valid for a particular
// crawler. The crawler must determine the correct group of records by
// finding the group with the most specific user-agent that still
// matches. All other groups of records are ignored by the crawler. The
// user-agent is non-case-sensitive. The order of the groups within the
// robots.txt file is irrelevant.
func (r *Robots) FindGroup(agent string) *Group {
	var prefixLen int
	var ret *Group

	agent = strings.ToLower(agent)
	for _, g := range r.groups {
		for _, a := range g.agents {
			if a == "*" && prefixLen == 0 {
				// Weakest match possible
				prefixLen = 1
				ret = g
			} else if strings.HasPrefix(agent, a) {
				if l := len(a); l > prefixLen {
					prefixLen = l
					ret = g
				}
			}
		}
	}

	if ret == nil {
		return emptyGroup
	}
	return ret
}

func (g *Group) Test(path string) bool {
	if r := g.findRule(path); r != nil {
		return r.allow
	}
	// When no rule applies, allow by default
	return true
}

// From Google's spec:
// The path value is used as a basis to determine whether or not a rule
// applies to a specific URL on a site. With the exception of wildcards,
// the path is used to match the beginning of a URL (and any valid URLs
// that start with the same path).
//
// At a group-member level, in particular for allow and disallow
// directives, the most specific rule based on the length of the [path]
// entry will trump the less specific (shorter) rule. The order of
// precedence for rules with wildcards is undefined.
func (g *Group) findRule(path string) (ret *rule) {
	var prefixLen int

	for _, r := range g.rules {
		if r.pattern != nil {
			if r.pattern.MatchString(path) {
				// Consider this a match equal to the length of the
				// pattern. From Google's spec:
				// The order of precedence for rules with wildcards is
				// undefined.
				if l := len(r.pattern.String()); l > prefixLen {
					prefixLen = len(r.pattern.String())
					ret = r
				}
			}
		} else if r.path == "/" && prefixLen == 0 {
			// Weakest match possible
			prefixLen = 1
			ret = r
		} else if strings.HasPrefix(path, r.path) {
			if l := len(r.path); l > prefixLen {
				prefixLen = l
				ret = r
			}
		}
	}
	return
}
