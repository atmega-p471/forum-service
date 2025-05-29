package domain

import (
	"testing"
	"time"
)

func TestMessage_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		expected bool
	}{
		{
			name: "Non-expired message",
			message: &Message{
				ID:        1,
				UserID:    1,
				Username:  "testuser",
				Content:   "Test content",
				CreatedAt: time.Now().Add(-1 * time.Hour),
				IsBanned:  false,
			},
			expected: false,
		},
		{
			name: "Banned message should be considered expired",
			message: &Message{
				ID:        1,
				UserID:    1,
				Username:  "testuser",
				Content:   "Test content",
				CreatedAt: time.Now().Add(-1 * time.Hour),
				IsBanned:  true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.message.IsBanned
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComment_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		comment  *Comment
		expected bool
	}{
		{
			name: "Non-expired comment",
			comment: &Comment{
				ID:        1,
				MessageID: 1,
				UserID:    1,
				Username:  "testuser",
				Content:   "Test comment",
				CreatedAt: time.Now().Add(-1 * time.Hour),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Expired comment",
			comment: &Comment{
				ID:        1,
				MessageID: 1,
				UserID:    1,
				Username:  "testuser",
				Content:   "Test comment",
				CreatedAt: time.Now().Add(-2 * time.Hour),
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.comment.IsExpired()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
	}{
		{
			name: "Valid message",
			message: &Message{
				UserID:   1,
				Username: "testuser",
				Content:  "Valid content",
			},
			wantErr: false,
		},
		{
			name: "Empty content",
			message: &Message{
				UserID:   1,
				Username: "testuser",
				Content:  "",
			},
			wantErr: true,
		},
		{
			name: "Empty username",
			message: &Message{
				UserID:   1,
				Username: "",
				Content:  "Valid content",
			},
			wantErr: true,
		},
		{
			name: "Content too long",
			message: &Message{
				UserID:   1,
				Username: "testuser",
				Content:  string(make([]byte, 1001)), // 1001 characters
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
