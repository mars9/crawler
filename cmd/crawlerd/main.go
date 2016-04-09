package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/mars9/crawler/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

func main() {
	var (
		addr = flag.String("addr", "localhost:8456", "crawler service network listen address")
		tls  = flag.Bool("tls", false, "connection uses TLS if specified, else plain TCP")
		cert = flag.String("cert", "", "TLS certificate file")
		key  = flag.String("key", "", "TLS private key file")

		opts []grpc.ServerOption
		self = os.Args[0]
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", self)
		fmt.Fprint(os.Stderr, usageMsg)
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if len(os.Args) <= 1 {
		flag.Usage()
	}
	flag.Parse()

	if *tls {
		creds, err := credentials.NewServerTLSFromFile(*cert, *key)
		if err != nil {
			grpclog.Fatalf("generating credentials: %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	done := make(chan error, 1)
	go func(addr string, done chan<- error, opts ...grpc.ServerOption) {
		done <- rpc.ListenAndServe("tcp", addr, opts...)
	}(*addr, done, opts...)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	select {
	case err := <-done:
		if err != nil {
			grpclog.Fatal(err)
		}
	case <-sig:
		// nothing
	}
}

const usageMsg = ``
