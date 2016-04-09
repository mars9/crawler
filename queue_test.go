package crawler

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

func TestQueueBasic(t *testing.T) {
	got, want := 0, 2000
	q := NewQueue(0, time.Second*30)

	done := make(chan struct{})
	go func() {
		for _ = range q.Pop() {
			got++
		}
		close(done)
	}()

	for i := 0; i < want; i++ {
		u, _ := url.Parse(fmt.Sprintf("https://golang.org/page%d", i))
		if err := q.Push(u); err != nil {
			t.Fatalf("send url: %v", err)
		}
	}
	if err := q.Close(); err != nil {
		t.Fatalf("close queue: expected <nil> error, got %v", err)
	}

	<-done

	if got != want {
		t.Fatalf("queue: expected %d results, got %d", want, got)
	}
}

func TestQueueBasicTimeout(t *testing.T) {
	got, want := 0, 2000
	q := NewQueue(0, time.Millisecond*10)

	done := make(chan struct{})
	go func() {
		for _ = range q.Pop() {
			got++
		}
		close(done)
	}()

	for i := 0; i < want; i++ {
		u, _ := url.Parse(fmt.Sprintf("https://golang.org/page%d", i))
		if err := q.Push(u); err != nil {
			t.Fatalf("send url: %v", err)
		}
	}

	<-done

	if got != want {
		t.Fatalf("queue: expected %d results, got %d", want, got)
	}
}

func TestQueueClose(t *testing.T) {
	q := NewQueue(0, time.Second*30)

	if err := q.Close(); err != nil {
		t.Fatalf("close queue: expected <nil> error, got %v", err)
	}
	if err := q.Close(); err != ErrQueueClosed {
		t.Fatalf("close queue: expected %v error, got %v", ErrQueueClosed, err)
	}
	if err := q.Push(nil); err != ErrQueueClosed {
		t.Fatalf("send queue: expected %v error, got %v", ErrQueueClosed, err)
	}

	if _, ok := <-q.push; ok {
		t.Fatalf("close queue: expected closed push channel")
	}
	if _, ok := <-q.pop; ok {
		t.Fatalf("close queue: expected closed pop channel")
	}

	q = NewQueue(8, time.Millisecond*1) // test queue timeout
	time.Sleep(time.Millisecond * 3)

	if err := q.Push(nil); err != ErrQueueClosed {
		t.Fatalf("send queue: expected %v error, got %v", ErrQueueClosed, err)
	}
	if err := q.Close(); err != ErrQueueClosed {
		t.Fatalf("close queue: expected %v error, got %v", ErrQueueClosed, err)
	}

	if _, ok := <-q.push; ok {
		t.Fatalf("close queue: expected closed push channel")
	}
	if _, ok := <-q.pop; ok {
		t.Fatalf("close queue: expected closed pop channel")
	}
}
