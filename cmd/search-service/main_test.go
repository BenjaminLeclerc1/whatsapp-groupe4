package main

import (
	"testing"
	"time"
)

func TestTokenize(t *testing.T) {
	// tokenize garde uniquement [a-z0-9] comme lettres (pas les accents après split).
	tests := []struct {
		input    string
		expected int
	}{
		{"Hello world", 2},
		{"Bonjour, comment ça va?", 4}, // bonjour, comment, a, va
		{"Le chat est noir", 4},        // le, chat, est, noir
		{"", 0},
		{"a", 1},
	}

	for _, test := range tests {
		result := tokenize(test.input)
		if len(result) != test.expected {
			t.Errorf("tokenize(%q) returned %d words, expected %d. Words: %v",
				test.input, len(result), test.expected, result)
		}
	}
}

func TestTokenizeNormalization(t *testing.T) {
	result := tokenize("HELLO World!")
	expected := []string{"hello", "world"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d words, got %d", len(expected), len(result))
	}

	for i, word := range result {
		if word != expected[i] {
			t.Errorf("Expected word %q at position %d, got %q", expected[i], i, word)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !contains(slice, "banana") {
		t.Error("Expected contains to return true for 'banana'")
	}

	if contains(slice, "orange") {
		t.Error("Expected contains to return false for 'orange'")
	}
}

func TestRemoveString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry", "banana"}
	result := removeString(slice, "banana")

	if len(result) != 2 {
		t.Errorf("Expected 2 items after removal, got %d", len(result))
	}

	if contains(result, "banana") {
		t.Error("'banana' should have been removed from slice")
	}
}

func TestCreateHighlight(t *testing.T) {
	content := "This is a very long message that contains many words and we want to test the highlight functionality"
	queryWords := []string{"message", "contains"}

	highlight := createHighlight(content, queryWords)

	if highlight == "" {
		t.Error("Highlight should not be empty")
	}

	if len(highlight) > len(content) {
		t.Error("Highlight should not be longer than original content")
	}

	t.Logf("Highlight: %s", highlight)
}

func TestMessageCreation(t *testing.T) {
	now := time.Now()
	message := Message{
		ID:        "msg-123",
		SenderID:  "user-456",
		Content:   "Test message",
		ChatID:    "chat-789",
		CreatedAt: now,
		Status:    "sent",
	}

	if message.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got '%s'", message.ID)
	}

	if message.Content != "Test message" {
		t.Errorf("Expected Content 'Test message', got '%s'", message.Content)
	}
}

func TestSearchResultScore(t *testing.T) {
	result := SearchResult{
		Message: Message{
			ID:      "msg-1",
			Content: "Hello world",
		},
		Score:     5,
		Highlight: "Hello world",
	}

	if result.Score != 5 {
		t.Errorf("Expected score 5, got %d", result.Score)
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("SEARCH_TEST_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}
