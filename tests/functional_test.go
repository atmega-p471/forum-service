package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// FunctionalTestSuite contains the functional tests for the forum service
type FunctionalTestSuite struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewFunctionalTestSuite creates a new functional test suite
func NewFunctionalTestSuite(baseURL string) *FunctionalTestSuite {
	return &FunctionalTestSuite{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Helper function to make HTTP requests
func (suite *FunctionalTestSuite) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, suite.BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return suite.HTTPClient.Do(req)
}

// TestFunctional_MessageWorkflow tests the complete message workflow
func TestFunctional_MessageWorkflow(t *testing.T) {
	// Skip if no base URL provided
	baseURL := "http://localhost:8082"
	suite := NewFunctionalTestSuite(baseURL)

	t.Run("Create Message", func(t *testing.T) {
		payload := map[string]interface{}{
			"content": "Functional test message",
		}

		resp, err := suite.makeRequest("POST", "/api/v1/messages", payload)
		if err != nil {
			t.Skipf("Skipping functional test - service not available: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := result["id"]; !ok {
			t.Error("Response should contain message ID")
		}

		if content, ok := result["content"].(string); !ok || content != "Functional test message" {
			t.Error("Response should contain correct content")
		}
	})

	t.Run("List Messages", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/v1/messages?limit=10&offset=0", nil)
		if err != nil {
			t.Skipf("Skipping functional test - service not available: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := result["messages"]; !ok {
			t.Error("Response should contain messages array")
		}

		if _, ok := result["total"]; !ok {
			t.Error("Response should contain total count")
		}
	})

	t.Run("Create and Get Comments", func(t *testing.T) {
		// First create a message
		messagePayload := map[string]interface{}{
			"content": "Message for comment test",
		}

		messageResp, err := suite.makeRequest("POST", "/api/v1/messages", messagePayload)
		if err != nil {
			t.Skipf("Skipping functional test - service not available: %v", err)
		}
		defer messageResp.Body.Close()

		if messageResp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create message: status %d", messageResp.StatusCode)
		}

		var messageResult map[string]interface{}
		err = json.NewDecoder(messageResp.Body).Decode(&messageResult)
		if err != nil {
			t.Fatalf("Failed to decode message response: %v", err)
		}

		messageID := messageResult["id"].(float64)

		// Create a comment
		commentPayload := map[string]interface{}{
			"content": "Functional test comment",
		}

		commentEndpoint := fmt.Sprintf("/api/v1/messages/%.0f/comments", messageID)
		commentResp, err := suite.makeRequest("POST", commentEndpoint, commentPayload)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		defer commentResp.Body.Close()

		if commentResp.StatusCode != http.StatusCreated {
			t.Errorf("Expected comment creation status 201, got %d", commentResp.StatusCode)
		}

		// Get comments
		getCommentsResp, err := suite.makeRequest("GET", commentEndpoint, nil)
		if err != nil {
			t.Fatalf("Failed to get comments: %v", err)
		}
		defer getCommentsResp.Body.Close()

		if getCommentsResp.StatusCode != http.StatusOK {
			t.Errorf("Expected get comments status 200, got %d", getCommentsResp.StatusCode)
		}

		var comments []map[string]interface{}
		err = json.NewDecoder(getCommentsResp.Body).Decode(&comments)
		if err != nil {
			t.Fatalf("Failed to decode comments response: %v", err)
		}

		if len(comments) == 0 {
			t.Error("Expected at least one comment")
		}

		if len(comments) > 0 {
			if content, ok := comments[0]["content"].(string); !ok || content != "Functional test comment" {
				t.Error("Comment should have correct content")
			}
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Test creating message with empty content
		payload := map[string]interface{}{
			"content": "",
		}

		resp, err := suite.makeRequest("POST", "/api/v1/messages", payload)
		if err != nil {
			t.Skipf("Skipping functional test - service not available: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty content, got %d", resp.StatusCode)
		}

		// Test getting comments for non-existent message
		resp, err = suite.makeRequest("GET", "/api/v1/messages/99999/comments", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Should return empty array or 404
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404 for non-existent message, got %d", resp.StatusCode)
		}
	})
}

// TestFunctional_CommentDeletion tests comment deletion functionality
func TestFunctional_CommentDeletion(t *testing.T) {
	baseURL := "http://localhost:8082"
	suite := NewFunctionalTestSuite(baseURL)

	// Create a message first
	messagePayload := map[string]interface{}{
		"content": "Message for deletion test",
	}

	messageResp, err := suite.makeRequest("POST", "/api/v1/messages", messagePayload)
	if err != nil {
		t.Skipf("Skipping functional test - service not available: %v", err)
	}
	defer messageResp.Body.Close()

	if messageResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create message: status %d", messageResp.StatusCode)
	}

	var messageResult map[string]interface{}
	err = json.NewDecoder(messageResp.Body).Decode(&messageResult)
	if err != nil {
		t.Fatalf("Failed to decode message response: %v", err)
	}

	messageID := messageResult["id"].(float64)

	// Create a comment
	commentPayload := map[string]interface{}{
		"content": "Comment to be deleted",
	}

	commentEndpoint := fmt.Sprintf("/api/v1/messages/%.0f/comments", messageID)
	commentResp, err := suite.makeRequest("POST", commentEndpoint, commentPayload)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}
	defer commentResp.Body.Close()

	if commentResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create comment: status %d", commentResp.StatusCode)
	}

	var commentResult map[string]interface{}
	err = json.NewDecoder(commentResp.Body).Decode(&commentResult)
	if err != nil {
		t.Fatalf("Failed to decode comment response: %v", err)
	}

	commentID := commentResult["id"].(float64)

	// Delete the comment
	deleteEndpoint := fmt.Sprintf("/api/v1/comments/%.0f", commentID)
	deleteResp, err := suite.makeRequest("DELETE", deleteEndpoint, nil)
	if err != nil {
		t.Fatalf("Failed to delete comment: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		t.Errorf("Expected delete status 200, got %d", deleteResp.StatusCode)
	}

	// Verify comment is deleted by trying to delete again
	deleteResp2, err := suite.makeRequest("DELETE", deleteEndpoint, nil)
	if err != nil {
		t.Fatalf("Failed to make second delete request: %v", err)
	}
	defer deleteResp2.Body.Close()

	if deleteResp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for already deleted comment, got %d", deleteResp2.StatusCode)
	}
}

// TestFunctional_HealthCheck tests service health
func TestFunctional_HealthCheck(t *testing.T) {
	baseURL := "http://localhost:8082"
	suite := NewFunctionalTestSuite(baseURL)

	resp, err := suite.makeRequest("GET", "/health", nil)
	if err != nil {
		t.Skipf("Skipping functional test - service not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected health check status 200, got %d", resp.StatusCode)
	}
}

// TestFunctional_CORS tests CORS headers
func TestFunctional_CORS(t *testing.T) {
	baseURL := "http://localhost:8082"
	suite := NewFunctionalTestSuite(baseURL)

	// Test preflight request
	req, err := http.NewRequest("OPTIONS", baseURL+"/api/v1/messages", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Origin", "http://localhost:8000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	resp, err := suite.HTTPClient.Do(req)
	if err != nil {
		t.Skipf("Skipping functional test - service not available: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
}

// TestFunctional_Pagination tests message pagination
func TestFunctional_Pagination(t *testing.T) {
	baseURL := "http://localhost:8082"
	suite := NewFunctionalTestSuite(baseURL)

	// Create multiple messages
	for i := 0; i < 5; i++ {
		payload := map[string]interface{}{
			"content": fmt.Sprintf("Pagination test message %d", i+1),
		}

		resp, err := suite.makeRequest("POST", "/api/v1/messages", payload)
		if err != nil {
			t.Skipf("Skipping functional test - service not available: %v", err)
		}
		resp.Body.Close()
	}

	// Test pagination
	resp, err := suite.makeRequest("GET", "/api/v1/messages?limit=2&offset=0", nil)
	if err != nil {
		t.Skipf("Skipping functional test - service not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	messages, ok := result["messages"].([]interface{})
	if !ok {
		t.Fatal("Response should contain messages array")
	}

	if len(messages) > 2 {
		t.Errorf("Expected at most 2 messages with limit=2, got %d", len(messages))
	}

	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("Response should contain total count")
	}

	if total < 5 {
		t.Errorf("Expected total >= 5 messages, got %.0f", total)
	}
}
