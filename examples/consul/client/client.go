package main

import (
	"time"
	"log"
	"google.golang.org/grpc"
	"context"
	"google.golang.org/grpc/balancer/roundrobin"
	"fmt"
	"github.com/nosixtools/grpc-go-plugins/discovery/consul"
	"github.com/nosixtools/grpc-go-plugins/examples/consul/proto"
)

func main() {
	opts := consul.NewResolverOpts("127.0.0.1:8500", "HelloService", "", nil)

	schema, err := consul.GenerateAndRegisterConsulResolver(opts)
	if err != nil {
		log.Fatal("init consul resovler err", err.Error())
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(fmt.Sprintf("%s:///HelloService", schema), grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := proto.NewHelloServiceClient(conn)

	// Contact the server and print out its response.
	name := "nosixtools"

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := c.SayHello(ctx, &proto.HelloRequest{Name: name})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("Hello: %s", r.Result)
		time.Sleep(time.Second)
	}

}
