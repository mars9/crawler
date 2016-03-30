package crawler

import (
	"net/url"
	"path"
	"strings"
	"sync"
)

func normalize(parent *url.URL, href string) (*url.URL, error) {
	candidate, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	if candidate.IsAbs() {
		return candidate, nil
	}

	href = strings.TrimSpace(href)
	switch {
	case len(href) > 0 && href[0] == '#':
		href = parent.Scheme + "://" + join(parent.Host, parent.Path) + href

	case strings.HasPrefix(href, "//"):
		href = parent.Scheme + ":" + href

	case len(href) > 0 && href[0] == '/':
		href = parent.Scheme + "://" + parent.Host + href

	default:
		href = "/" + href
		href = parent.Scheme + "://" + join(parent.Host, parent.Path) + href
	}
	return url.Parse(href) // verify, normalize url
}

func join(host, name string) string {
	for len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if len(name) == 1 && name[0] == '.' {
		return ""
	}
	if len(name) > 1 {
		name = path.Clean(name)
	}
	if len(name) == 0 {
		return host
	}
	return path.Join(host, name)
}

type index struct {
	mu sync.Mutex // we check and set, hence we cannot use sync.RWMutex
	m  map[string]struct{}
}

func newIndex() *index { return &index{m: make(map[string]struct{})} }

func (i *index) normalizeKey(url *url.URL) string {
	name := url.Path

	if len(name) == 1 && name[0] == '.' {
		name = ""
	}
	if len(name) > 1 {
		name = path.Clean(name)
	}
	if len(name) > 0 && name[0] != '/' {
		name = "/" + name
	}
	if len(name) == 0 {
		name = "/"
	}

	if len(url.RawQuery) > 0 {
		name += "?" + url.RawQuery
	}
	return name
}

func (i *index) Has(url *url.URL) bool {
	key := i.normalizeKey(url)
	i.mu.Lock()
	_, found := i.m[key]
	if !found {
		i.m[key] = struct{}{}
	}
	i.mu.Unlock()
	return found
}
