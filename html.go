package crawler

import (
	"bytes"

	"golang.org/x/net/html"
)

func parseHTML(data []byte) (*html.Node, error) {
	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return node, nil
}
