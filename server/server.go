package main

import (
	"io"
	"log"

	pb "github.com/movaua/grpc-messenger/contract"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewMessengerServer(responseBroadcaster ResponseBroadcastServer) pb.MessengerServer {
	return &messengerServer{
		responseBroadcaster: responseBroadcaster}
}

type messengerServer struct {
	responseBroadcaster ResponseBroadcastServer
	stream              pb.Messenger_ChatServer
	pb.UnimplementedMessengerServer
}

func (s *messengerServer) Chat(stream pb.Messenger_ChatServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "failed to get metadata")
	}

	userValues := md.Get("x-user")
	if len(userValues) != 1 {
		return status.Errorf(codes.Unauthenticated, "failed to get user")
	}
	user := userValues[0]

	log.Printf("user connected: %s", user)
	defer log.Printf("user disconnected: %s", user)

	send := s.responseBroadcaster.Subscribe()
	defer func() {
		s.responseBroadcaster.CancelSubscription(send)
	}()

	doneCh := make(chan struct{})
	errCh := make(chan error)

	go func() {
		for {
			select {
			case response := <-send:
				log.Printf("sending response to user %s: %v", user, response)
				if err := stream.Send(response); err != nil {
					log.Printf("could not send response to user %s: %v", user, err)
					errCh <- err
					return
				}
			case <-stream.Context().Done():
				log.Printf("stopped sending response to user %s", user)
				return
			}
		}
	}()

	go func() {
		for {
			request, err := stream.Recv()

			if err == io.EOF {
				var done struct{}
				doneCh <- done
				return
			}

			if err != nil {
				errCh <- err
				return
			}

			response := &pb.Response{
				User: user,
				Text: request.Text,
			}

			s.responseBroadcaster.Broadcast(response)
		}
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			return nil
		case <-stream.Context().Done():
			return nil
		}
	}
}
