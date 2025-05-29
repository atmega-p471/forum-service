package repository

import (
	"database/sql"
	"errors"
	"os"

	"github.com/forum/forum-service/internal/domain"
	_ "github.com/mattn/go-sqlite3"
)

// Repository encapsulates all repositories
type Repository struct {
	Message domain.MessageRepository
}

// NewRepository creates a new repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Message: NewMessageRepository(db),
	}
}

// InitSchema initializes the database schema
func InitSchema(db *sql.DB) error {
	// Enable foreign key support
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		return err
	}

	// Create messages table (only if it doesn't exist)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			is_banned BOOLEAN NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}

	// Create comments table (only if it doesn't exist)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes for better performance
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_comments_message_id ON comments(message_id)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_comments_expires_at ON comments(expires_at)`)
	if err != nil {
		return err
	}

	// Verify tables were created
	var messageTableExists, commentTableExists bool
	err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&messageTableExists)
	if err != nil {
		return err
	}
	err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='comments'").Scan(&commentTableExists)
	if err != nil {
		return err
	}

	if !messageTableExists || !commentTableExists {
		return errors.New("failed to create tables")
	}

	return nil
}
