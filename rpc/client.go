package rpc

import (
	"fmt"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/mars9/crawler/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client struct {
	client proto.CrawlerClient
	conn   *grpc.ClientConn
}

func Dial(network, addr string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	c := proto.NewCrawlerClient(conn)

	return &Client{client: c, conn: conn}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

type Service string

const (
	Start Service = "start"
	Stop  Service = "stop"
)

type Call struct {
	Service Service // service name
	Args    protobuf.Message
	Reply   protobuf.Message
	err     error
	ch      chan<- *Call
}

func (c Call) Err() error { return c.err }

func (c *Call) done() {
	select {
	case c.ch <- c:
	default:
		panic("cannot send response message")
	}
}

func (c *Client) Call(ctx context.Context, call *Call, ch chan<- *Call) {
	call.ch = ch

	switch call.Service {
	case Start:
		req, ok := call.Args.(*proto.StartRequest)
		if !ok {
			call.err = fmt.Errorf("invalid request type: %T", call.Args)
		}
		call.Reply, call.err = c.client.Start(ctx, req)
	case Stop:
		req, ok := call.Args.(*proto.StopRequest)
		if !ok {
			call.err = fmt.Errorf("invalid request type: %T", call.Args)
		}
		call.Reply, call.err = c.client.Stop(ctx, req)
	default:
		call.err = fmt.Errorf("unknown service: %q", call.Service)
	}

	call.done()
}
