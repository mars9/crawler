package rpc

import (
	"errors"
	"net"
	"net/url"
	"sync"
	"time"

	crawler "github.com/mars9/crawler"
	"github.com/mars9/crawler/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	errAlreadyRegistered = errors.New("crawler already registered")
	errNotRegistered     = errors.New("crawler not registered")
)

type call struct {
	cancel context.CancelFunc
	c      *crawler.Crawler
}

type server struct {
	mu   sync.RWMutex // protects following
	call map[string]*call
}

func newServer() *server { return &server{call: make(map[string]*call)} }

func (s *server) Start(ctx context.Context, req *proto.StartRequest) (*proto.StartResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &proto.StartResponse{Timestamp: time.Now().Unix()}
	w, err := newRequest(req)
	if err != nil {
		return resp, err
	}

	if _, found := s.hasCall(w.Host.Host); found {
		return resp, errAlreadyRegistered
	}

	seeds, workers := parseSeeds(req.Seeds), verifyWorkers(req.Workers)
	sitemap, err := url.Parse(req.Sitemap)
	if err != nil {
		return resp, err
	}
	if !sitemap.IsAbs() {
		sitemap = nil // ignore invalid sitemap URL
	}

	if sitemap == nil && len(seeds) == 0 {
		return resp, errors.New("invalid sitemap and no valid seeds")
	}

	nctx, cancel := context.WithCancel(ctx)
	call := &call{
		c:      crawler.New(w, workers, time.Duration(req.TimeToLive)),
		cancel: cancel,
	}
	s.putCall(w.Host.Host, call)
	go s.callWaiter(nctx, w.Host.Host, call.c)

	err = call.c.Start(sitemap, seeds...)
	resp.Workers = int32(workers)
	return resp, err
}

func (s *server) callWaiter(ctx context.Context, host string, c *crawler.Crawler) {
	select {
	case <-ctx.Done():
		grpclog.Printf("crawler %q canceled: %v", host, ctx.Err())
	case <-c.Done():
		grpclog.Printf("crawler %q done", host)
	}

	s.mu.Lock() // remove host anyway
	s.deleteCall(host)
	s.mu.Unlock()
}

func (s *server) Stop(ctx context.Context, req *proto.StopRequest) (*proto.StopResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &proto.StopResponse{Timestamp: time.Now().Unix()}
	host, err := url.Parse(req.Hostname)
	if err != nil {
		return resp, err
	}

	call, found := s.hasCall(host.Host)
	if !found {
		return resp, errNotRegistered
	}
	call.cancel()
	s.deleteCall(host.Host)

	return resp, err
}

func (s *server) putCall(host string, c *call) {
	s.call[host] = c
}

func (s *server) hasCall(host string) (*call, bool) {
	call, found := s.call[host]
	return call, found
}

func (s *server) deleteCall(host string) {
	delete(s.call, host)
}

func ListenAndServe(network, addr string, opts ...grpc.ServerOption) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	return Serve(listener, opts...)
}

func Serve(listener net.Listener, opts ...grpc.ServerOption) error {
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterCrawlerServer(grpcServer, newServer())
	return grpcServer.Serve(listener)
}
