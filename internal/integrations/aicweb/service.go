package aicweb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Service 抽象
type Service interface {
	Register(ctx context.Context, req *RegisterRequest) error
	Login(ctx context.Context, req *LoginRequest) (token string, err error)
	Validate(ctx context.Context, token string) (*user, error)
}

type memoryService struct {
	mu           sync.RWMutex
	usersByEmail map[string]*user  // email -> user
	tokens       map[string]string // token -> email
}

func NewServiceMemory() Service {
	return &memoryService{
		usersByEmail: map[string]*user{},
		tokens:       map[string]string{},
	}
}

func (s *memoryService) Register(ctx context.Context, req *RegisterRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.usersByEmail[req.Email]; ok {
		return ErrEmailAlreadyUse
	}

	u := &user{
		ID:        randID(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // 仅开发演示：明文；正式环境请使用 hash
		CreatedAt: time.Now(),
	}
	s.usersByEmail[req.Email] = u
	return nil
}

func (s *memoryService) Login(ctx context.Context, req *LoginRequest) (string, error) {
	s.mu.RLock()
	u, ok := s.usersByEmail[req.Email]
	s.mu.RUnlock()

	if !ok || u.Password != req.Password {
		return "", ErrUnauthorized
	}

	t := randID()
	s.mu.Lock()
	s.tokens[t] = req.Email
	s.mu.Unlock()
	return t, nil
}

func (s *memoryService) Validate(ctx context.Context, token string) (*user, error) {
	s.mu.RLock()
	email, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrUnauthorized
	}

	s.mu.RLock()
	u := s.usersByEmail[email]
	s.mu.RUnlock()
	if u == nil {
		return nil, ErrUnauthorized
	}
	return u, nil
}

func randID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
