package gormrepository

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type loggedSessionRepository struct {
	db *gorm.DB
}

// Create implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) Create(session *model.LoggedSession) error {
	return l.db.Create(session).Error
}

// GetByRefreshTokenHash implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) GetByRefreshTokenHash(refreshTokenHash string) (*model.LoggedSession, error) {
	var session model.LoggedSession
	err := l.db.Where("refresh_token_hash = ? AND is_revoked = ? AND expiry_at > ?",
		refreshTokenHash, false, time.Now().Unix()).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// GetByUserID implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) GetByUserID(userID uuid.UUID) ([]*model.LoggedSession, error) {
	var sessions []*model.LoggedSession
	err := l.db.Where("user_id = ?", userID).Find(&sessions).Error
	return sessions, err
}

// GetActiveSessionsByUserID implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) GetActiveSessionsByUserID(userID uuid.UUID) ([]*model.LoggedSession, error) {
	var sessions []*model.LoggedSession
	err := l.db.Where("user_id = ? AND is_revoked = ? AND expiry_at > ?",
		userID, false, time.Now().Unix()).Find(&sessions).Error
	return sessions, err
}

// Update implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) Update(session *model.LoggedSession) error {
	return l.db.Save(session).Error
}

// Delete implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) Delete(id uuid.UUID) error {
	return l.db.Delete(&model.LoggedSession{}, id).Error
}

// DeleteByUserID implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) DeleteByUserID(userID uuid.UUID) error {
	return l.db.Where("user_id = ?", userID).Delete(&model.LoggedSession{}).Error
}

// RevokeSession implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) RevokeSession(id uuid.UUID) error {
	return l.db.Model(&model.LoggedSession{}).Where("id = ?", id).Update("is_revoked", true).Error
}

// RevokeAllUserSessions implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) RevokeAllUserSessions(userID uuid.UUID) error {
	return l.db.Model(&model.LoggedSession{}).Where("user_id = ?", userID).Update("is_revoked", true).Error
}

// CleanExpiredSessions implements repository.LoggedSessionRepository.
func (l *loggedSessionRepository) CleanExpiredSessions() error {
	return l.db.Where("expiry_at < ? OR is_revoked = ?", time.Now().Unix(), true).Delete(&model.LoggedSession{}).Error
}

func newLoggedSessionRepository(db *gorm.DB) irepository.LoggedSessionRepository {
	return &loggedSessionRepository{db: db}
}
