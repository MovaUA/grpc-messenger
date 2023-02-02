package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	pb "github.com/movaua/grpc-messenger/contract"
	"github.com/movaua/grpc-messenger/server/broadcast"
	"github.com/movaua/grpc-messenger/server/messenger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	port                = flag.Int("port", 8080, "The server port")
	responseBroadcaster broadcast.ResponseBroadcastServer
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	ctx := NewAppContext()
	responseBroadcaster = broadcast.NewResponseBroadcastServer(ctx)

	var opts []grpc.ServerOption
	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     120 * time.Second,
		MaxConnectionAge:      1<<63 - 1, // infinity
		MaxConnectionAgeGrace: 1<<63 - 1, // infinity
		Time:                  60 * time.Second,
		Timeout:               30 * time.Second,
	}))
	opts = append(opts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		MinTime:             9 * time.Second,
		PermitWithoutStream: false,
	}))

	s := grpc.NewServer(opts...)
	pb.RegisterMessengerServer(s, messenger.NewMessengerServer(responseBroadcaster))

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func NewAppContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	defer func() {
		signal.Stop(c)
	}()

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx
}
