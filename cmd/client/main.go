package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	todopb "grpc-todo/proto_gen"
)

func main() {
	addr := flag.String("addr", ":50051", "server address")
	flag.Parse()

	conn, err := grpc.NewClient(
		*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	client := todopb.NewTodoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(flag.Args()) > 0 {
		runCmd(ctx, client, flag.Args())
		return
	}

	repl(ctx, client)
}

func runCmd(ctx context.Context, client todopb.TodoServiceClient, args []string) {
	switch args[0] {
	case "create":
		if len(args) < 2 {
			log.Fatal("usage: create <title> [description]")
		}
		desc := ""
		if len(args) > 2 {
			desc = args[2]
		}
		todo, err := client.CreateTodo(ctx, &todopb.CreateTodoRequest{
			Title:       args[1],
			Description: desc,
		})
		if err != nil {
			log.Fatalf("create failed: %v", err)
		}
		fmt.Printf("created: %+v\n", todo)

	case "get":
		if len(args) < 2 {
			log.Fatal("usage: get <id>")
		}
		todo, err := client.GetTodo(ctx, &todopb.GetTodoRequest{Id: args[1]})
		if err != nil {
			log.Fatalf("get failed: %v", err)
		}
		fmt.Printf("todo: %+v\n", todo)

	case "list":
		resp, err := client.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			log.Fatalf("list failed: %v", err)
		}
		fmt.Printf("page %d/%d (size=%d, total=%d):\n", resp.Page, resp.TotalPages, resp.PageSize, resp.TotalCount)
		for _, t := range resp.Todos {
			fmt.Printf("  [%s] %s (completed=%v)\n", t.Id, t.Title, t.Completed)
		}

	case "update":
		if len(args) < 3 {
			log.Fatal("usage: update <id> <title> [completed]")
		}
		completed := false
		if len(args) > 3 && args[3] == "true" {
			completed = true
		}
		todo, err := client.UpdateTodo(ctx, &todopb.UpdateTodoRequest{
			Id:        args[1],
			Title:     args[2],
			Completed: completed,
		})
		if err != nil {
			log.Fatalf("update failed: %v", err)
		}
		fmt.Printf("updated: %+v\n", todo)

	case "delete":
		if len(args) < 2 {
			log.Fatal("usage: delete <id>")
		}
		_, err := client.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: args[1]})
		if err != nil {
			log.Fatalf("delete failed: %v", err)
		}
		fmt.Println("deleted")

	default:
		log.Fatalf("unknown command: %s (create|get|list|update|delete)", args[0])
	}
}

func repl(ctx context.Context, client todopb.TodoServiceClient) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("gRPC Todo client. Commands: create|get|list|update|delete|help|quit")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		args := strings.Fields(line)
		switch args[0] {
		case "quit", "exit":
			return
		case "help":
			fmt.Println(`Commands:
  create <title> [description]
  get <id>
  list [page] [page_size]
  update <id> <title> [completed]
  delete <id>
  quit`)
		default:
			runCmd(ctx, client, args)
		}
	}
}
