package usecase

import (
	"time"

	"github.com/forum/forum-service/internal/config"
	"github.com/forum/forum-service/internal/delivery/grpc/client"
	"github.com/forum/forum-service/internal/delivery/ws"
	"github.com/forum/forum-service/internal/domain"
	"github.com/forum/forum-service/internal/repository"
)

// UseCase implements domain.MessageUseCase
type UseCase struct {
	repo       domain.MessageRepository
	authClient *client.AuthClient
	hub        *ws.Hub
}

// GetMessages implements domain.MessageUseCase
func (u *UseCase) GetMessages(limit, offset int64) ([]*domain.Message, int64, error) {
	return u.repo.List(limit, offset)
}

// CreateMessage implements domain.MessageUseCase
func (u *UseCase) CreateMessage(userID int64, username string, content string) (*domain.Message, error) {
	message := &domain.Message{
		UserID:   userID,
		Username: username,
		Content:  content,
		IsBanned: false,
	}
	id, err := u.repo.Create(message)
	if err != nil {
		return nil, err
	}
	message.ID = id
	return message, nil
}

// BanMessage implements domain.MessageUseCase
func (u *UseCase) BanMessage(id int64) error {
	return u.repo.Ban(id)
}

// UnbanMessage implements domain.MessageUseCase
func (u *UseCase) UnbanMessage(id int64) error {
	return u.repo.Unban(id)
}

// GetByID implements domain.MessageUseCase
func (u *UseCase) GetByID(id int64) (*domain.Message, error) {
	return u.repo.GetByID(id)
}

// CreateComment implements domain.MessageUseCase
func (u *UseCase) CreateComment(messageID, userID int64, username, content string) (*domain.Comment, error) {
	comment := &domain.Comment{
		MessageID: messageID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
	}

	id, err := u.repo.CreateComment(comment)
	if err != nil {
		return nil, err
	}

	comment.ID = id
	return comment, nil
}

// GetComments implements domain.MessageUseCase
func (u *UseCase) GetComments(messageID int64) ([]*domain.Comment, error) {
	return u.repo.GetComments(messageID)
}

// GetAllMessages implements domain.MessageUseCase
func (u *UseCase) GetAllMessages() ([]*domain.Message, error) {
	return u.repo.GetAllMessages()
}

// DeleteMessage implements domain.MessageUseCase
func (u *UseCase) DeleteMessage(id int64) error {
	return u.repo.Delete(id)
}

// DeleteComment implements domain.MessageUseCase
func (u *UseCase) DeleteComment(id int64) error {
	return u.repo.DeleteComment(id)
}

// NewUseCase creates a new usecase
func NewUseCase(repo *repository.Repository, authClient *client.AuthClient, hub *ws.Hub, cfg *config.Config) domain.MessageUseCase {
	return &UseCase{
		repo:       repo.Message,
		authClient: authClient,
		hub:        hub,
	}
}
