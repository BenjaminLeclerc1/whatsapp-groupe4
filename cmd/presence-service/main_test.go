package main

import (
	"testing"
	"time"
)

func TestPresenceStatus(t *testing.T) {
	// Test des constantes de statut
	if StatusOnline != "online" {
		t.Errorf("Expected StatusOnline to be 'online', got '%s'", StatusOnline)
	}
	if StatusOffline != "offline" {
		t.Errorf("Expected StatusOffline to be 'offline', got '%s'", StatusOffline)
	}
	if StatusTyping != "typing" {
		t.Errorf("Expected StatusTyping to be 'typing', got '%s'", StatusTyping)
	}
}

func TestPresenceCreation(t *testing.T) {
	now := time.Now()
	presence := Presence{
		UserID:       "user-123",
		Status:       StatusOnline,
		LastActivity: now,
		LastSeen:     now,
	}

	if presence.UserID != "user-123" {
		t.Errorf("Expected UserID to be 'user-123', got '%s'", presence.UserID)
	}
	if presence.Status != StatusOnline {
		t.Errorf("Expected Status to be 'online', got '%s'", presence.Status)
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("NONEXISTENT_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", result)
	}
}

func TestTimeouts(t *testing.T) {
	// Vérifier que les constantes de timeout sont définies
	if activityTimeout != 5*time.Minute {
		t.Errorf("Expected activityTimeout to be 5 minutes, got %v", activityTimeout)
	}
	if typingTimeout != 10*time.Second {
		t.Errorf("Expected typingTimeout to be 10 seconds, got %v", typingTimeout)
	}
}
