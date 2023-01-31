package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	pb "github.com/movaua/grpc-messenger/contract"
	"google.golang.org/grpc"
)

var (
	port                = flag.Int("port", 8080, "The server port")
	responseBroadcaster ResponseBroadcastServer
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	ctx := NewAppContext()
	responseBroadcaster = NewResponseBroadcastServer(ctx)

	s := grpc.NewServer()
	pb.RegisterMessengerServer(s, NewMessengerServer(responseBroadcaster))

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
