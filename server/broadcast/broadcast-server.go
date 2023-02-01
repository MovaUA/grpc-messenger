package broadcast

import (
	"context"
	"log"

	pb "github.com/movaua/grpc-messenger/contract"
)

func NewResponseBroadcastServer(ctx context.Context) ResponseBroadcastServer {
	var server = &responseBroadcastServer{
		source:         make(chan *pb.Response),
		addListener:    make(chan chan *pb.Response),
		removeListener: make(chan (<-chan *pb.Response)),
	}
	go server.serve(ctx)
	return server
}

type ResponseBroadcastServer interface {
	Subscribe() <-chan *pb.Response
	CancelSubscription(<-chan *pb.Response)
	Broadcast(response *pb.Response)
}

type responseBroadcastServer struct {
	source         chan *pb.Response
	listeners      []chan *pb.Response
	addListener    chan chan *pb.Response
	removeListener chan (<-chan *pb.Response)
}

func (s *responseBroadcastServer) Subscribe() <-chan *pb.Response {
	log.Println("subscribing...")
	defer log.Println("subscribed")

	var newListener = make(chan *pb.Response)
	s.addListener <- newListener
	return newListener
}

func (s *responseBroadcastServer) CancelSubscription(listener <-chan *pb.Response) {
	log.Println("cancelling subscription...")

	s.removeListener <- listener

	log.Println("subscription cancelled")
}

func (s *responseBroadcastServer) serve(ctx context.Context) {
	defer func() {
		for _, listener := range s.listeners {
			close(listener)
		}
		log.Printf("stopped serving %T\n", s)
	}()

	log.Printf("serving %T\n", s)

	for {
		select {
		case <-ctx.Done():
			return
		case newListener := <-s.addListener:
			s.listeners = append(s.listeners, newListener)
		case listenerToRemove := <-s.removeListener:
			for i, ch := range s.listeners {
				if ch == listenerToRemove {
					s.listeners[i] = s.listeners[len(s.listeners)-1]
					s.listeners = s.listeners[:len(s.listeners)-1]
					close(ch)
					break
				}
			}
		case response, ok := <-s.source:
			if !ok {
				return
			}
			log.Printf("broadcasting... %v", response)
			for _, listener := range s.listeners {
				select {
				case listener <- response:
				case <-ctx.Done():
					return
				}
			}
			log.Printf("broadcasted %v", response)
		}
	}
}

func (s *responseBroadcastServer) Broadcast(response *pb.Response) {
	s.source <- response
}
