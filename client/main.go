package main

import (
	"log"

	"../api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Authentication holds the login/password
type Authentication struct {
	Login    string
	Password string
}

// GetRequestMetadata gets the current request metadata
func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"login":    a.Login,
		"password": a.Password,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security
func (a *Authentication) RequireTransportSecurity() bool {
	return true
}

// instantiates a client connection, on the TCP port the server is bound to
func main() {
	var conn *grpc.ClientConn

	// Create the client TLS credentials
	// Client does not use the certificate key, the key is private to the server
	creds, err := credentials.NewClientTLSFromFile("cert/server.crt", "")
	if err != nil {
		log.Fatalf("could not load tls cert: %s", err)
	}

	// Setup login/pass
	auth := Authentication{
		Login:    "john",
		Password: "doe",
	}

	// Initiate a connection with the server
	// Dial variadic function, so it accepts any number of functions
	conn, err = grpc.Dial("localhost:7777",
		grpc.WithTransportCredentials(creds),
		// function takes an interface as a parameter, Authentication structure should
		// comply to that interface (getRequestMetadata, and RequireTransportSecurity)
		grpc.WithPerRPCCredentials(&auth))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	// close the connection when the function returns
	defer conn.Close()

	// Client for the Ping service, that calls the SayHello function, passing
	// a PingMessage to it
	c := api.NewPingClient(conn)

	response, err := c.SayHello(context.Background(), &api.PingMessage{Greeting: "foo"})
	if err != nil {
		log.Fatalf("Error when calling SayHello: %s", err)
	}
	log.Printf("Response from server: %s", response.Greeting)
}
