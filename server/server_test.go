package server

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"grpc-todo/proto_gen"
)

func setupTest(t *testing.T) (todopb.TodoServiceServer, context.Context) {
	t.Helper()
	store := NewInMemoryStore()
	svc := NewService(store)
	return svc, context.Background()
}

func TestCreateTodo(t *testing.T) {
	svc, ctx := setupTest(t)

	t.Run("valid", func(t *testing.T) {
		todo, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{
			Title:       "test",
			Description: "desc",
		})
		if err != nil {
			t.Fatal(err)
		}
		if todo.Title != "test" {
			t.Errorf("got title %q, want %q", todo.Title, "test")
		}
		if todo.Id == "" {
			t.Error("expected non-empty id")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{Title: ""})
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("got code %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})
}

func TestGetTodo(t *testing.T) {
	svc, ctx := setupTest(t)

	created, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{Title: "my todo"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetTodo(ctx, &todopb.GetTodoRequest{Id: created.Id})
		if err != nil {
			t.Fatal(err)
		}
		if got.Title != "my todo" {
			t.Errorf("got title %q, want %q", got.Title, "my todo")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetTodo(ctx, &todopb.GetTodoRequest{Id: "nonexistent"})
		if status.Code(err) != codes.NotFound {
			t.Errorf("got code %v, want %v", status.Code(err), codes.NotFound)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := svc.GetTodo(ctx, &todopb.GetTodoRequest{Id: ""})
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("got code %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})
}

func TestListTodos(t *testing.T) {
	svc, ctx := setupTest(t)

	for i := range 5 {
		_, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{
			Title: fmt.Sprintf("todo %d", i+1),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("first page", func(t *testing.T) {
		resp, err := svc.ListTodos(ctx, &todopb.ListTodosRequest{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Todos) != 2 {
			t.Errorf("got %d todos, want 2", len(resp.Todos))
		}
		if resp.TotalCount != 5 {
			t.Errorf("got total %d, want 5", resp.TotalCount)
		}
	})

	t.Run("default pagination", func(t *testing.T) {
		resp, err := svc.ListTodos(ctx, &todopb.ListTodosRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Todos) != 5 {
			t.Errorf("got %d todos, want 5", len(resp.Todos))
		}
	})
}

func TestUpdateTodo(t *testing.T) {
	svc, ctx := setupTest(t)

	created, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{Title: "original"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update title and completed", func(t *testing.T) {
		updated, err := svc.UpdateTodo(ctx, &todopb.UpdateTodoRequest{
			Id:        created.Id,
			Title:     "updated",
			Completed: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Title != "updated" {
			t.Errorf("got title %q, want %q", updated.Title, "updated")
		}
		if !updated.Completed {
			t.Error("expected completed=true")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.UpdateTodo(ctx, &todopb.UpdateTodoRequest{Id: "nonexistent"})
		if status.Code(err) != codes.NotFound {
			t.Errorf("got code %v, want %v", status.Code(err), codes.NotFound)
		}
	})
}

func TestDeleteTodo(t *testing.T) {
	svc, ctx := setupTest(t)

	created, err := svc.CreateTodo(ctx, &todopb.CreateTodoRequest{Title: "to delete"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("delete existing", func(t *testing.T) {
		_, err := svc.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: created.Id})
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.GetTodo(ctx, &todopb.GetTodoRequest{Id: created.Id})
		if status.Code(err) != codes.NotFound {
			t.Errorf("got code %v, want %v", status.Code(err), codes.NotFound)
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		_, err := svc.DeleteTodo(ctx, &todopb.DeleteTodoRequest{Id: "nonexistent"})
		if status.Code(err) != codes.NotFound {
			t.Errorf("got code %v, want %v", status.Code(err), codes.NotFound)
		}
	})
}
