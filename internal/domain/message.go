package domain

import (
	"errors"
	"strings"
	"time"
)

// Message represents a message entity
type Message struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	IsBanned  bool      `json:"is_banned"`
}

// Validate validates the message
func (m *Message) Validate() error {
	if strings.TrimSpace(m.Content) == "" {
		return errors.New("content cannot be empty")
	}
	if strings.TrimSpace(m.Username) == "" {
		return errors.New("username cannot be empty")
	}
	if len(m.Content) > 1000 {
		return errors.New("content too long")
	}
	return nil
}

// Comment represents a comment entity
type Comment struct {
	ID        int64     `json:"id"`
	MessageID int64     `json:"message_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired checks if the comment has expired
func (c *Comment) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Validate validates the comment
func (c *Comment) Validate() error {
	if strings.TrimSpace(c.Content) == "" {
		return errors.New("content cannot be empty")
	}
	if strings.TrimSpace(c.Username) == "" {
		return errors.New("username cannot be empty")
	}
	if len(c.Content) > 500 {
		return errors.New("comment too long")
	}
	if c.MessageID <= 0 {
		return errors.New("invalid message ID")
	}
	return nil
}

// MessageRepository defines the repository interface for Message
type MessageRepository interface {
	GetByID(id int64) (*Message, error)
	List(limit, offset int64) ([]*Message, int64, error)
	GetAllMessages() ([]*Message, error)
	Create(message *Message) (int64, error)
	Ban(id int64) error
	Unban(id int64) error
	Delete(id int64) error
	CreateComment(comment *Comment) (int64, error)
	GetComments(messageID int64) ([]*Comment, error)
	GetCommentByID(id int64) (*Comment, error)
	DeleteComment(id int64) error
	DeleteExpiredComments() error
}

// MessageUseCase defines the usecase interface for Message
type MessageUseCase interface {
	GetMessages(limit, offset int64) ([]*Message, int64, error)
	GetAllMessages() ([]*Message, error)
	CreateMessage(userID int64, username, content string) (*Message, error)
	BanMessage(id int64) error
	UnbanMessage(id int64) error
	GetByID(id int64) (*Message, error)
	CreateComment(messageID, userID int64, username, content string) (*Comment, error)
	GetComments(messageID int64) ([]*Comment, error)
	DeleteMessage(id int64) error
	DeleteComment(id int64) error
}

// User represents a minimal user structure for forum service
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsBanned bool   `json:"is_banned"`
}
