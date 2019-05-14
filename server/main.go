package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	// allows for service handles and the Server struct to be available
	"../api"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// private type for Context keys
type contextKey int

const (
	clientIDKey contextKey = iota
)

// credMatch function to match the credentials header,
// allowing them to be metadat in the gRPC context. This is how we have all the authentication working,
// because the reverse-proxy uses the HTTP headers it receives when it connects to the gRPC server
func credMatcher(headerName string) (mdName string, ok bool) {
	if headerName == "Login" || headerName == "Password" {
		return headerName, true
	}
	return "", false
}

// authenticateAgent check the client credentials
func authenticateClient(ctx context.Context, s *api.Server) (string, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		clientLogin := strings.Join(md["login"], "")
		clientPassword := strings.Join(md["password"], "")

		if clientLogin != "john" {
			return "", fmt.Errorf("uknown user %s", clientLogin)
		}
		if clientPassword != "doe" {
			return "", fmt.Errorf("bad password %s", clientPassword)
		}

		log.Printf("authenticated client: %s", clientLogin)
		return "42", nil
	}

	return "", fmt.Errorf("missing credentials")
}

// unary Interceptor calls authenticateClient with current context
/*
	context.Context object, containing your data, and that will exist during all the lifetime of the request
	interface{} inbound parameter of the RPC call
	UnaryServerInfo struct which contains a bunch of information about the call
	UnaryHandler struct which is the handler invoked by UnaryServerInterceptor to complete normal execution of a unaryRPC
*/
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	s, ok := info.Server.(*api.Server)
	if !ok {
		return nil, fmt.Errorf("unable to cast server")
	}

	// unaryInterceptor makes sure the grpc.UnaryServerInfo has the right server abstraction, and call the
	// authentication function, authenticateClient
	clientID, err := authenticateClient(ctx, s)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, clientIDKey, clientID)
	return handler(ctx, req)
}

func startGRPCServer(address, certFile, keyFile string) error {
	// create a listener on TCP port
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// create a server instance
	s := api.Server{}

	// Create the TLS credentials object from certificate and key files
	// you need to precisely specify the IP you bind your server to, so
	// that the IP matches the FQDN used in the certificate.
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("could not load TLS keys: %s", err)
	}

	// Create an array of gRPC options with the credentials
	opts := []grpc.ServerOption{grpc.Creds(creds),
		grpc.UnaryInterceptor(unaryInterceptor)}

	// create a gRPC server object
	// create a gRPC server object
	// variadic function, you can pass it any number of trailing arguments
	grpcServer := grpc.NewServer(opts...)

	// attach the Ping service to the server
	api.RegisterPingServer(grpcServer, &s)

	// start the server
	log.Printf("starting HTTP/2 gRPC server on %s", address)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %s", err)
	}

	return nil
}

// create the REST gateway
func startRESTServer(address, grpcAddress, certFile string) error {
	// you start by getting the context.Context background object
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Then create a multiplexer object, mux, with an option: runtime.WithIncomingHeaderMatcher
	// This option takes a function reference as a parameter, credMatch, and is called for every
	// HTTP header from the incoming request. The function evaluates whether or not the
	// HTTP header should be passed to the gRPC context
	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(credMatcher))

	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return fmt.Errorf("could not load TLS certificate: %s", err)
	}

	// Setup the client gRPC options
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	// Register ping
	// register your api endpoint, make the link between multiplexer, gRPC server, using
	// the context and the gRPC options
	err = api.RegisterPingHandlerFromEndpoint(ctx, mux, grpcAddress, opts)
	if err != nil {
		return fmt.Errorf("could not register service Ping: %s", err)
	}

	log.Printf("starting HTTP/1.1 REST server on %s", address)

	http.ListenAndServe(address, mux)

	return nil
}

// main start a gRPC server and waits for connection
func main() {
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", 7777)
	restAddress := fmt.Sprintf("%s:%d", "localhost", 7778)
	certFile := "cert/server.crt"
	keyFile := "cert/server.key"

	// fire the gRPC server in a goroutine
	go func() {
		err := startGRPCServer(grpcAddress, certFile, keyFile)
		if err != nil {
			log.Fatalf("failed to start gRPC server: %s", err)
		}
	}()

	// fire the REST server in a goroutine
	go func() {
		err := startRESTServer(restAddress, grpcAddress, certFile)
		if err != nil {
			log.Fatalf("failed to start gRPC server: %s", err)
		}
	}()

	// infinite loop
	log.Printf("Entering infinite loop")

	// Blocking select call, so that the program does not end right away
	select {}
}
