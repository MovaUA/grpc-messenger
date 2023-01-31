package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/movaua/grpc-messenger/contract"
)

var (
	useTLS = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	port   = flag.Int("port", 8080, "The server port")
	user   = flag.String("user", "anonym", "The user name")
)

func main() {
	flag.Parse()

	ctx := createAppContext()

	var opts []grpc.DialOption

	var transportCredentials credentials.TransportCredentials
	if *useTLS {
		transportCredentials = credentials.NewTLS(&tls.Config{})
	} else {
		transportCredentials = insecure.NewCredentials()
	}
	opts = append(opts, grpc.WithTransportCredentials(transportCredentials))

	// Anything linked to this variable will transmit request headers.
	md := metadata.New(map[string]string{"x-user": *user})
	ctx = metadata.NewOutgoingContext(ctx, md)

	serverAddress := fmt.Sprintf("localhost:%d", *port)
	conn, err := grpc.Dial(serverAddress, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMessengerClient(conn)

	chat, err := client.Chat(ctx)
	if err != nil {
		log.Fatalf("could not chat: %v", err)
	}

	go func() {
		for {
			response, err := chat.Recv()
			if err != nil {
				log.Fatalf("could not receive: %v", err)
			}
			fmt.Printf("%s: %s\n", response.User, response.Text)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var text = scanner.Text()
		err = chat.Send(&pb.Request{Text: text})
		if err != nil {
			log.Fatalf("could not send: %v", err)
		}
	}
}

func createAppContext() context.Context {
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
