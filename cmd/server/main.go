package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"grpc-todo/proto_gen"
	"grpc-todo/server"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC server listen address")
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	store := server.NewInMemoryStore()
	svc := server.NewService(store)

	srv := grpc.NewServer()
	todopb.RegisterTodoServiceServer(srv, svc)
	reflection.Register(srv)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		srv.GracefulStop()
	}()

	log.Printf("gRPC server listening on %s", *addr)
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
