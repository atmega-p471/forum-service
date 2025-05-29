package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/atmega-p471/forum-service/internal/domain"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize schema
	err = InitSchema(db)
	if err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	return db
}

func TestMessageRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)

	message := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Test message content",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	id, err := repo.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	// Verify message was created
	created, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("Failed to get created message: %v", err)
	}

	if created.Username != message.Username {
		t.Errorf("Expected username %s, got %s", message.Username, created.Username)
	}
	if created.Content != message.Content {
		t.Errorf("Expected content %s, got %s", message.Content, created.Content)
	}
}

func TestMessageRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)

	// Test non-existent message
	_, err := repo.GetByID(999)
	if err == nil {
		t.Error("Expected error for non-existent message")
	}

	// Create and test existing message
	message := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Test message content",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	id, err := repo.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	retrieved, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("Failed to get message by ID: %v", err)
	}

	if retrieved.ID != id {
		t.Errorf("Expected ID %d, got %d", id, retrieved.ID)
	}
}

func TestMessageRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)

	// Create test messages
	for i := 0; i < 5; i++ {
		message := &domain.Message{
			UserID:    int64(i + 1),
			Username:  "testuser" + string(rune(i+'1')),
			Content:   "Test message " + string(rune(i+'1')),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
			IsBanned:  false,
		}
		_, err := repo.Create(message)
		if err != nil {
			t.Fatalf("Failed to create test message: %v", err)
		}
	}

	// Test list with limit and offset
	messages, total, err := repo.List(3, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Test with offset
	messages, _, err = repo.List(3, 3)
	if err != nil {
		t.Fatalf("Failed to list messages with offset: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestMessageRepository_Ban(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)

	// Create test message
	message := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Test message content",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	id, err := repo.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Ban the message
	err = repo.Ban(id)
	if err != nil {
		t.Fatalf("Failed to ban message: %v", err)
	}

	// Verify message is banned
	banned, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("Failed to get banned message: %v", err)
	}

	if !banned.IsBanned {
		t.Error("Expected message to be banned")
	}
}

func TestMessageRepository_CreateComment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)

	// Create test message first
	message := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Test message content",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	messageID, err := repo.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create comment
	comment := &domain.Comment{
		MessageID: messageID,
		UserID:    2,
		Username:  "commenter",
		Content:   "Test comment",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	commentID, err := repo.CreateComment(comment)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}

	if commentID <= 0 {
		t.Errorf("Expected positive comment ID, got %d", commentID)
	}

	// Verify comment was created
	comments, err := repo.GetComments(messageID)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}

	if comments[0].Content != comment.Content {
		t.Errorf("Expected comment content %s, got %s", comment.Content, comments[0].Content)
	}
}
