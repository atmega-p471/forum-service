package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/forum/forum-service/internal/domain"
)

// MessageRepository is a message repository
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sql.DB) domain.MessageRepository {
	return &MessageRepository{
		db: db,
	}
}

// GetByID gets a message by ID
func (r MessageRepository) GetByID(id int64) (*domain.Message, error) {
	var message domain.Message
	var createdAt string

	err := r.db.QueryRow("SELECT id, user_id, username, content, created_at, is_banned FROM messages WHERE id = ?", id).
		Scan(&message.ID, &message.UserID, &message.Username, &message.Content, &createdAt, &message.IsBanned)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}

	message.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &message, nil
}

// List gets a list of messages
func (r MessageRepository) List(limit, offset int64) ([]*domain.Message, int64, error) {
	// First, get the total count
	var total int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the messages
	rows, err := r.db.Query("SELECT id, user_id, username, content, created_at, is_banned FROM messages ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var message domain.Message
		var createdAt string

		err := rows.Scan(&message.ID, &message.UserID, &message.Username, &message.Content, &createdAt, &message.IsBanned)
		if err != nil {
			return nil, 0, err
		}

		message.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, 0, err
		}
		messages = append(messages, &message)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetAllMessages gets all messages (admin only)
func (r MessageRepository) GetAllMessages() ([]*domain.Message, error) {
	rows, err := r.db.Query("SELECT id, user_id, username, content, created_at, is_banned FROM messages ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var message domain.Message
		var createdAt string

		err := rows.Scan(&message.ID, &message.UserID, &message.Username, &message.Content, &createdAt, &message.IsBanned)
		if err != nil {
			return nil, err
		}

		message.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// Create creates a new message
func (r MessageRepository) Create(message *domain.Message) (int64, error) {
	message.CreatedAt = time.Now().UTC()
	res, err := r.db.Exec("INSERT INTO messages (user_id, username, content, created_at, is_banned) VALUES (?, ?, ?, ?, ?)",
		message.UserID, message.Username, message.Content, message.CreatedAt.Format(time.RFC3339), message.IsBanned)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Ban bans a message
func (r MessageRepository) Ban(id int64) error {
	_, err := r.db.Exec("UPDATE messages SET is_banned = 1 WHERE id = ?", id)
	return err
}

// Unban unbans a message
func (r MessageRepository) Unban(id int64) error {
	_, err := r.db.Exec("UPDATE messages SET is_banned = 0 WHERE id = ?", id)
	return err
}

// CreateComment creates a new comment
func (r MessageRepository) CreateComment(comment *domain.Comment) (int64, error) {
	// First check if the message exists
	_, err := r.GetByID(comment.MessageID)
	if err != nil {
		return 0, err
	}

	comment.CreatedAt = time.Now().UTC()
	comment.ExpiresAt = comment.CreatedAt.Add(5 * time.Minute) // Comments expire after 5 minutes

	res, err := r.db.Exec("INSERT INTO comments (message_id, user_id, username, content, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
		comment.MessageID, comment.UserID, comment.Username, comment.Content,
		comment.CreatedAt.Format(time.RFC3339), comment.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetComments gets all comments for a message (excluding expired ones)
func (r MessageRepository) GetComments(messageID int64) ([]*domain.Comment, error) {
	// First check if the message exists
	_, err := r.GetByID(messageID)
	if err != nil {
		return nil, err
	}

	// Only get comments that haven't expired yet
	now := time.Now().UTC()
	rows, err := r.db.Query("SELECT id, message_id, user_id, username, content, created_at, expires_at FROM comments WHERE message_id = ? AND expires_at > ? ORDER BY created_at ASC", messageID, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*domain.Comment
	for rows.Next() {
		var comment domain.Comment
		var createdAt, expiresAt string

		err := rows.Scan(&comment.ID, &comment.MessageID, &comment.UserID, &comment.Username, &comment.Content, &createdAt, &expiresAt)
		if err != nil {
			return nil, err
		}

		comment.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, err
		}
		comment.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

// Delete deletes a message completely (admin only)
func (r MessageRepository) Delete(id int64) error {
	// First delete all comments for this message
	_, err := r.db.Exec("DELETE FROM comments WHERE message_id = ?", id)
	if err != nil {
		return err
	}

	// Then delete the message
	_, err = r.db.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

// GetCommentByID gets a comment by ID
func (r MessageRepository) GetCommentByID(id int64) (*domain.Comment, error) {
	var comment domain.Comment
	var createdAt, expiresAt string

	err := r.db.QueryRow("SELECT id, message_id, user_id, username, content, created_at, expires_at FROM comments WHERE id = ?", id).
		Scan(&comment.ID, &comment.MessageID, &comment.UserID, &comment.Username, &comment.Content, &createdAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("comment not found")
		}
		return nil, err
	}

	comment.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	comment.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	return &comment, nil
}

// DeleteComment deletes a comment completely (admin only)
func (r MessageRepository) DeleteComment(id int64) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	return err
}

// DeleteExpiredComments deletes all expired comments
func (r MessageRepository) DeleteExpiredComments() error {
	now := time.Now().UTC()
	_, err := r.db.Exec("DELETE FROM comments WHERE expires_at <= ?", now.Format(time.RFC3339))
	return err
}
