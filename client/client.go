package main // import "go.dedis.ch/eonet/client"

import (
	"context"
	"log"
	"time"

	"go.dedis.ch/example-grpc/count"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	address = "localhost:3000"
)

func main() {
	// Read the certificate created by the server.
	creds, err := credentials.NewClientTLSFromFile("cert.pem", "")
	if err != nil {
		log.Fatal(err)
	}

	// Connection using TLS.
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Make a request to the server.
	c := count.NewCountClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Println("Counting...")

	r, err := c.Count(ctx, &count.CountRequest{Value: 0})
	if err != nil {
		log.Fatalf("could not count: %v", err)
	}
	log.Printf("Count: %d\n", r.GetValue())
	log.Printf("Certificates: %d\n", len(r.GetCertificates()))
}
