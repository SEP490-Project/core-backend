package repository

import (
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type LoggedSessionRepository interface {
	Create(session *model.LoggedSession) error
	GetByRefreshTokenHash(refreshTokenHash string) (*model.LoggedSession, error)
	GetByUserID(userID uuid.UUID) ([]*model.LoggedSession, error)
	GetActiveSessionsByUserID(userID uuid.UUID) ([]*model.LoggedSession, error)
	Update(session *model.LoggedSession) error
	Delete(id uuid.UUID) error
	DeleteByUserID(userID uuid.UUID) error
	RevokeSession(id uuid.UUID) error
	RevokeAllUserSessions(userID uuid.UUID) error
	CleanExpiredSessions() error
}
