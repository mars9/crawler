// Copyright 2016 Markus Sonderegger. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crawler

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"runtime"
	"testing"
)

func assert(t *testing.T, prefix string, expected, got interface{}) {
	if e, ok := expected.([]byte); ok {
		if bytes.Compare(e, got.([]byte)) != 0 {
			_, file, line, _ := runtime.Caller(1)
			fmt.Printf("\t%s:%d expected %s %q, got %q\n",
				path.Base(file), line, prefix, expected, got)
			t.FailNow()
		}
	}

	if !reflect.DeepEqual(expected, got) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\t%s:%d expected %s %#v, got %#v\n",
			path.Base(file), line, prefix, expected, got)
		t.FailNow()
	}
}

func TestQueue(t *testing.T) {
	t.Parallel()

	pushc := make(chan *url.URL, 3)
	popc := make(chan *url.URL, 3)
	donec := make(chan struct{})
	want := make([]*url.URL, 10)
	var got []*url.URL
	for i := 0; i < 10; i++ {
		want[i], _ = url.Parse(fmt.Sprintf("http://example.com/site%d", i))
	}

	go Queue(pushc, popc)
	go func() {
		for url := range popc {
			got = append(got, url)
		}
		donec <- struct{}{}
	}()

	for i := 0; i < 10; i++ {
		url, _ := url.Parse(fmt.Sprintf("http://example.com/site%d", i))
		pushc <- url
	}
	close(pushc)

	<-donec

	assert(t, "items", 10, len(got))
	for i := range want {
		assert(t, "url", want[i], got[i])
	}
	if _, ok := <-popc; ok {
		t.Fatalf("expected closed pop channel")
	}
}
