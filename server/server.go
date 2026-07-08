package server

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	todopb "grpc-todo/proto_gen"
)

type Store interface {
	Create(ctx context.Context, title, description string) (*todopb.Todo, error)
	Get(ctx context.Context, id string) (*todopb.Todo, error)
	List(ctx context.Context, page, pageSize int32) ([]*todopb.Todo, int32, error)
	Update(ctx context.Context, id, title, description string, completed bool) (*todopb.Todo, error)
	Delete(ctx context.Context, id string) error
}

type InMemoryStore struct {
	mu     sync.RWMutex
	todos  map[string]*todopb.Todo
	nextID int64
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		todos:  make(map[string]*todopb.Todo),
		nextID: 1,
	}
}

func (s *InMemoryStore) Create(_ context.Context, title, description string) (*todopb.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%d", s.nextID)
	s.nextID++
	now := timestamppb.New(time.Now().UTC())

	todo := &todopb.Todo{
		Id:          id,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.todos[id] = todo
	return todo, nil
}

func (s *InMemoryStore) Get(_ context.Context, id string) (*todopb.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "todo %q not found", id)
	}
	return todo, nil
}

func (s *InMemoryStore) List(_ context.Context, page, pageSize int32) ([]*todopb.Todo, int32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := int32(len(s.todos))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	start := (page - 1) * pageSize
	if start >= total {
		return nil, total, nil
	}

	end := min(start+pageSize, total)

	all := make([]*todopb.Todo, 0, len(s.todos))
	for _, t := range s.todos {
		all = append(all, t)
	}

	return all[start:end], total, nil
}

func (s *InMemoryStore) Update(_ context.Context, id, title, description string, completed bool) (*todopb.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "todo %q not found", id)
	}

	if title != "" {
		todo.Title = title
	}
	todo.Description = description
	todo.Completed = completed
	todo.UpdatedAt = timestamppb.New(time.Now().UTC())
	return todo, nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.todos[id]; !ok {
		return status.Errorf(codes.NotFound, "todo %q not found", id)
	}
	delete(s.todos, id)
	return nil
}

type Service struct {
	todopb.UnimplementedTodoServiceServer
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.Todo, error) {
	if req.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "title is required")
	}
	return s.store.Create(ctx, req.Title, req.Description)
}

func (s *Service) GetTodo(ctx context.Context, req *todopb.GetTodoRequest) (*todopb.Todo, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}
	return s.store.Get(ctx, req.Id)
}

func (s *Service) ListTodos(ctx context.Context, req *todopb.ListTodosRequest) (*todopb.ListTodosResponse, error) {
	page := max(req.GetPage(), 1)
	pageSize := req.GetPageSize()
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	todos, total, err := s.store.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	// Safely compute total pages without overflow.
	totalPages := int32(math.Ceil(float64(total) / float64(pageSize)))

	return &todopb.ListTodosResponse{
		Todos:      todos,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.Todo, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}
	return s.store.Update(ctx, req.Id, req.Title, req.Description, req.Completed)
}

func (s *Service) DeleteTodo(ctx context.Context, req *todopb.DeleteTodoRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}
	if err := s.store.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
