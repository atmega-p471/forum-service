package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/forum/forum-service/internal/domain"
	"github.com/gorilla/mux"
)

// MockMessageUseCase implements domain.MessageUseCase for testing
type MockMessageUseCase struct {
	messages map[int64]*domain.Message
	comments map[int64]*domain.Comment
	nextID   int64
}

func NewMockMessageUseCase() *MockMessageUseCase {
	return &MockMessageUseCase{
		messages: make(map[int64]*domain.Message),
		comments: make(map[int64]*domain.Comment),
		nextID:   1,
	}
}

func (m *MockMessageUseCase) GetMessages(limit, offset int64) ([]*domain.Message, int64, error) {
	var messages []*domain.Message
	var count int64

	for _, msg := range m.messages {
		if !msg.IsBanned {
			count++
			if count > offset && int64(len(messages)) < limit {
				messages = append(messages, msg)
			}
		}
	}

	return messages, count, nil
}

func (m *MockMessageUseCase) GetAllMessages() ([]*domain.Message, error) {
	var messages []*domain.Message
	for _, msg := range m.messages {
		messages = append(messages, msg)
	}
	return messages, nil
}

func (m *MockMessageUseCase) CreateMessage(userID int64, username, content string) (*domain.Message, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}

	id := m.nextID
	m.nextID++

	message := &domain.Message{
		ID:        id,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	m.messages[id] = message
	return message, nil
}

func (m *MockMessageUseCase) BanMessage(id int64) error {
	if msg, exists := m.messages[id]; exists {
		msg.IsBanned = true
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageUseCase) UnbanMessage(id int64) error {
	if msg, exists := m.messages[id]; exists {
		msg.IsBanned = false
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageUseCase) GetByID(id int64) (*domain.Message, error) {
	if msg, exists := m.messages[id]; exists {
		return msg, nil
	}
	return nil, errors.New("message not found")
}

func (m *MockMessageUseCase) CreateComment(messageID, userID int64, username, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}

	// Check if message exists
	if _, exists := m.messages[messageID]; !exists {
		return nil, errors.New("message not found")
	}

	id := m.nextID
	m.nextID++

	comment := &domain.Comment{
		ID:        id,
		MessageID: messageID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	m.comments[id] = comment
	return comment, nil
}

func (m *MockMessageUseCase) GetComments(messageID int64) ([]*domain.Comment, error) {
	var comments []*domain.Comment
	for _, comment := range m.comments {
		if comment.MessageID == messageID && !comment.IsExpired() {
			comments = append(comments, comment)
		}
	}
	return comments, nil
}

func (m *MockMessageUseCase) DeleteMessage(id int64) error {
	if _, exists := m.messages[id]; exists {
		delete(m.messages, id)
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageUseCase) DeleteComment(id int64) error {
	if _, exists := m.comments[id]; exists {
		delete(m.comments, id)
		return nil
	}
	return errors.New("comment not found")
}

func TestForumHandler_ListMessages(t *testing.T) {
	usecase := NewMockMessageUseCase()
	handler := NewForumHandler(usecase)

	// Create test messages
	_, err := usecase.CreateMessage(1, "user1", "Test message 1")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}
	_, err = usecase.CreateMessage(2, "user2", "Test message 2")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	req, err := http.NewRequest("GET", "/api/v1/messages?limit=10&offset=0", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ListMessages(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	messages, ok := response["messages"].([]interface{})
	if !ok {
		t.Error("Response should contain messages array")
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	total, ok := response["total"].(float64)
	if !ok {
		t.Error("Response should contain total count")
	}

	if int(total) != 2 {
		t.Errorf("Expected total 2, got %d", int(total))
	}
}

func TestForumHandler_CreateMessage(t *testing.T) {
	usecase := NewMockMessageUseCase()
	handler := NewForumHandler(usecase)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid message creation",
			payload: map[string]interface{}{
				"content": "Test message content",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Empty content",
			payload: map[string]interface{}{
				"content": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "No payload",
			payload:        nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.payload != nil {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatal(err)
				}
			}

			req, err := http.NewRequest("POST", "/api/v1/messages", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.CreateMessage(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if _, ok := response["id"]; !ok {
					t.Error("Response should contain message ID")
				}

				if content, ok := response["content"].(string); !ok || content != tt.payload["content"] {
					t.Error("Response should contain correct content")
				}
			}
		})
	}
}

func TestForumHandler_GetComments(t *testing.T) {
	usecase := NewMockMessageUseCase()
	handler := NewForumHandler(usecase)

	// Create test message
	message, err := usecase.CreateMessage(1, "user1", "Test message")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	// Create test comment
	_, err = usecase.CreateComment(message.ID, 2, "user2", "Test comment")
	if err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}

	req, err := http.NewRequest("GET", "/api/v1/messages/"+strconv.FormatInt(message.ID, 10)+"/comments", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Setup router to parse path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/messages/{id}/comments", handler.GetComments).Methods("GET")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var comments []map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &comments)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}

	if content, ok := comments[0]["content"].(string); !ok || content != "Test comment" {
		t.Error("Comment should have correct content")
	}
}

func TestForumHandler_CreateComment(t *testing.T) {
	usecase := NewMockMessageUseCase()
	handler := NewForumHandler(usecase)

	// Create test message
	message, err := usecase.CreateMessage(1, "user1", "Test message")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	tests := []struct {
		name           string
		messageID      string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name:      "Valid comment creation",
			messageID: strconv.FormatInt(message.ID, 10),
			payload: map[string]interface{}{
				"content": "Test comment content",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:      "Empty content",
			messageID: strconv.FormatInt(message.ID, 10),
			payload: map[string]interface{}{
				"content": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Invalid message ID",
			messageID: "999",
			payload: map[string]interface{}{
				"content": "Test comment",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest("POST", "/api/v1/messages/"+tt.messageID+"/comments", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Setup router to parse path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/messages/{id}/comments", handler.CreateComment).Methods("POST")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if _, ok := response["id"]; !ok {
					t.Error("Response should contain comment ID")
				}
			}
		})
	}
}

func TestForumHandler_DeleteComment(t *testing.T) {
	usecase := NewMockMessageUseCase()
	handler := NewForumHandler(usecase)

	// Create test message and comment
	message, err := usecase.CreateMessage(1, "user1", "Test message")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	comment, err := usecase.CreateComment(message.ID, 2, "user2", "Test comment")
	if err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}

	tests := []struct {
		name           string
		commentID      string
		expectedStatus int
	}{
		{
			name:           "Valid comment deletion",
			commentID:      strconv.FormatInt(comment.ID, 10),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existent comment",
			commentID:      "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/comments/"+tt.commentID, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Setup router to parse path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/comments/{id}", handler.DeleteComment).Methods("DELETE")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}
