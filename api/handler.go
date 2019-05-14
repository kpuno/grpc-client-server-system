package api

import (
	"log"

	"golang.org/x/net/context"
)

// Server represents the gRPC server
// abstraction of the server. It allows to "attach" some resources
// to your server, making them availble during the RPC calls
type Server struct{}

// SayHello generates response to a Ping request
// SayHello is defined in the Protobuf file, as the rpc call for the Ping
// service. If you don't define it, you won't be able to create the gRPC server

// Takes a PingMessage as a parameter, and returns a PingMessage
// PingMessage struct is defined in the api.pb.go file

// Context - https://blog.golang.org/context
func (s *Server) SayHello(ctx context.Context, in *PingMessage) (*PingMessage, error) {
	log.Printf("Receive message %s", in.Greeting)
	return &PingMessage{Greeting: "bar"}, nil
}
