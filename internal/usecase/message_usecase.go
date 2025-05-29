package usecase

import (
	"errors"
	"log"
	"time"

	"github.com/forum/forum-service/internal/delivery/grpc/client"
	"github.com/forum/forum-service/internal/domain"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrUserBanned      = errors.New("user is banned")
	ErrMessageTooLong  = errors.New("message is too long")
	ErrMessageEmpty    = errors.New("message cannot be empty")
	ErrInternalError   = errors.New("internal error")
)

// MessageUseCase implements domain.MessageUseCase
type MessageUseCase struct {
	repo       domain.MessageRepository
	authClient *client.AuthClient
	hub        Hub
}

// Hub defines a minimal interface for the WebSocket hub
type Hub interface {
	BroadcastMessage(*domain.Message)
}

// NewMessageUseCase creates a new message usecase
func NewMessageUseCase(repo domain.MessageRepository, authClient *client.AuthClient, hub Hub) domain.MessageUseCase {
	return &MessageUseCase{
		repo:       repo,
		authClient: authClient,
		hub:        hub,
	}
}

// GetMessages gets a list of messages
func (u *MessageUseCase) GetMessages(limit, offset int64) ([]*domain.Message, int64, error) {
	log.Printf("Getting messages with limit: %d, offset: %d", limit, offset)
	messages, total, err := u.repo.List(limit, offset)
	if err != nil {
		log.Printf("Error getting messages from repository: %v", err)
		return nil, 0, err
	}
	log.Printf("Successfully retrieved %d messages, total: %d", len(messages), total)
	return messages, total, nil
}

// GetAllMessages gets all messages (admin only)
func (u *MessageUseCase) GetAllMessages() ([]*domain.Message, error) {
	log.Printf("Getting all messages for admin")
	messages, err := u.repo.GetAllMessages()
	if err != nil {
		log.Printf("Error getting all messages from repository: %v", err)
		return nil, err
	}
	log.Printf("Successfully retrieved %d messages", len(messages))
	return messages, nil
}

// CreateMessage creates a new message
func (u *MessageUseCase) CreateMessage(userID int64, username, content string) (*domain.Message, error) {
	log.Printf("Creating message for user %d (%s)", userID, username)

	if content == "" {
		log.Printf("Empty content provided")
		return nil, errors.New("content is required")
	}

	// Skip auth validation for anonymous users (ID=0)
	if userID != 0 {
		// Validate user ID
		user, err := u.authClient.GetUser(userID)
		if err != nil {
			log.Printf("Error validating user: %v", err)
			return nil, err
		}

		// Check if user is banned
		if user.IsBanned {
			log.Printf("User %d is banned", userID)
			return nil, errors.New("user is banned")
		}
	}

	// Create message
	message := &domain.Message{
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		IsBanned:  false,
	}

	// Save message
	messageID, err := u.repo.Create(message)
	if err != nil {
		log.Printf("Error creating message in repository: %v", err)
		return nil, err
	}

	// Set message ID
	message.ID = messageID
	log.Printf("Successfully created message with ID: %d", messageID)

	// Broadcast message
	u.hub.BroadcastMessage(message)

	return message, nil
}

// BanMessage bans a message
func (u *MessageUseCase) BanMessage(id int64) error {
	// Check if message exists
	message, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}

	// Ban message
	err = u.repo.Ban(id)
	if err != nil {
		return err
	}

	// Update message
	message.IsBanned = true

	// Broadcast updated message
	u.hub.BroadcastMessage(message)

	return nil
}

// UnbanMessage unbans a message
func (u *MessageUseCase) UnbanMessage(id int64) error {
	// Check if message exists
	message, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}

	// Unban message
	err = u.repo.Unban(id)
	if err != nil {
		return err
	}

	// Update message
	message.IsBanned = false

	// Broadcast updated message
	u.hub.BroadcastMessage(message)

	return nil
}

func (u *MessageUseCase) GetByID(id int64) (*domain.Message, error) {
	message, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, ErrMessageNotFound
	}

	return message, nil
}

// CreateComment creates a new comment
func (u *MessageUseCase) CreateComment(messageID, userID int64, username, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}

	// Skip auth validation for anonymous users (ID=0)
	if userID != 0 {
		// Validate user ID
		user, err := u.authClient.GetUser(userID)
		if err != nil {
			return nil, err
		}

		// Check if user is banned
		if user.IsBanned {
			return nil, errors.New("user is banned")
		}
	}

	// Create comment
	comment := &domain.Comment{
		MessageID: messageID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Save comment
	commentID, err := u.repo.CreateComment(comment)
	if err != nil {
		return nil, err
	}

	// Set comment ID
	comment.ID = commentID

	return comment, nil
}

// GetComments gets all comments for a message
func (u *MessageUseCase) GetComments(messageID int64) ([]*domain.Comment, error) {
	return u.repo.GetComments(messageID)
}

// DeleteMessage deletes a message completely (admin only)
func (u *MessageUseCase) DeleteMessage(id int64) error {
	// Check if message exists
	message, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}

	// Delete message
	err = u.repo.Delete(id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteComment deletes a comment completely (admin only)
func (u *MessageUseCase) DeleteComment(id int64) error {
	// Check if comment exists
	comment, err := u.repo.GetCommentByID(id)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("comment not found")
	}

	// Delete comment
	err = u.repo.DeleteComment(id)
	if err != nil {
		return err
	}

	return nil
}

// CleanupExpiredComments removes all expired comments from the database
func (u *MessageUseCase) CleanupExpiredComments() error {
	log.Printf("Cleaning up expired comments...")
	err := u.repo.DeleteExpiredComments()
	if err != nil {
		log.Printf("Error cleaning up expired comments: %v", err)
		return err
	}
	log.Printf("Successfully cleaned up expired comments")
	return nil
}

// StartCleanupScheduler starts a background goroutine that periodically cleans up expired comments
func (u *MessageUseCase) StartCleanupScheduler() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		log.Printf("Started expired comments cleanup scheduler (checking every minute)")

		for {
			select {
			case <-ticker.C:
				if err := u.CleanupExpiredComments(); err != nil {
					log.Printf("Failed to cleanup expired comments: %v", err)
				}
			}
		}
	}()
}
