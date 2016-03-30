package crawler

import (
	"fmt"
	"log"
	"net/url"
	"testing"
	"time"
)

/*
func TestPool(t *testing.T) {
	b := &Balancer{pool: make(pool, 0, 10), table: make(map[int]int, 10)}
	heap.Init(&b.pool)

	for i := 0; i < 10; i++ {
		w := &worker{id: i, pending: 0}
		heap.Push(&b.pool, w)
		b.table[i] = w.pos
	}

	//	p.Increment(3)
	//	p.Increment(3)
	//	p.Increment(3)

	for _, w := range b.pool {
		fmt.Printf("%+v\n", w)
	}

	for i := 0; i < 20; i++ {
		b.dispatch()
		for _, w := range b.pool {
			fmt.Printf("\t%+v\n", w)
		}
	}
	/*
		b.dispatch()
		b.dispatch()
		b.dispatch()
		b.dispatch()
		b.dispatch()
		b.dispatch()
		b.dispatch()
	fmt.Println()
}
*/

func push(url *url.URL, prod *Producer) {
	log.Println("pushed", url)
	prod.Send(url)
}

func TestBalancer(t *testing.T) {
	var count int
	countTest := func(url *url.URL, prod *Producer) {
		//		log.Println("received", url)
		count++
		if count == 42 {
			log.Println("found")
			if err := prod.Close(); err != nil {
				t.Fatalf("close producer: %v", err)
			}
		}
	}
	b, prod := Start(countTest, 32, time.Second*10)

	for i := 0; i < 100; i++ {
		s := fmt.Sprintf("http://example.com/page%.4d", i)
		url, _ := url.Parse(s)
		prod.Send(url)
	}

	<-b.Done()
}
