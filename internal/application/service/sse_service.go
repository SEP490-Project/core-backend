package service

import (
	"fmt"
	"sync"

	"core-backend/internal/application/interfaces/iservice"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SSEService struct {
	clients map[uuid.UUID][]chan iservice.SSEMessage
	mu      sync.RWMutex
}

func NewSSEService() *SSEService {
	return &SSEService{
		clients: make(map[uuid.UUID][]chan iservice.SSEMessage),
	}
}

// SendUnreadCount implements iservice.RealTimeNotifier
func (s *SSEService) SendUnreadCount(userID uuid.UUID, count int64) error {
	return s.SendEvent(userID, "unread_count", fmt.Sprintf("%d", count))
}

// SendEvent implements iservice.RealTimeNotifier
func (s *SSEService) SendEvent(userID uuid.UUID, event string, data any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients, ok := s.clients[userID]
	if !ok {
		return nil // User not connected
	}

	msg := iservice.SSEMessage{
		Event: event,
		Data:  data,
	}

	for _, ch := range clients {
		select {
		case ch <- msg:
		default:
			zap.L().Warn("SSE client channel full, dropping message", zap.String("user_id", userID.String()))
		}
	}
	return nil
}

// Subscribe adds a client and returns a channel for messages
func (s *SSEService) Subscribe(userID uuid.UUID) (<-chan iservice.SSEMessage, func()) {
	clientChan := make(chan iservice.SSEMessage, 10)

	s.mu.Lock()
	s.clients[userID] = append(s.clients[userID], clientChan)
	s.mu.Unlock()

	zap.L().Info("SSE client connected", zap.String("user_id", userID.String()))

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		clients := s.clients[userID]
		for i, ch := range clients {
			if ch == clientChan {
				s.clients[userID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		if len(s.clients[userID]) == 0 {
			delete(s.clients, userID)
		}
		close(clientChan)
		zap.L().Info("SSE client disconnected", zap.String("user_id", userID.String()))
	}

	return clientChan, unsubscribe
}
