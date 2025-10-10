package auth

import (
	"context"
	"testing"
	"time"
)

func TestMemorySessionStore(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Test user
	user := &AuthUser{
		ID:       "test123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"user"},
	}

	// Test creating a session
	sessionID, err := store.CreateSession(ctx, user)
	if err != nil {
		t.Errorf("Failed to create session: %v", err)
	}

	if sessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	// Test retrieving the session
	retrievedUser, err := store.GetSession(ctx, sessionID)
	if err != nil {
		t.Errorf("Failed to get session: %v", err)
	}

	if retrievedUser == nil {
		t.Error("Expected user to be retrieved")
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, retrievedUser.Username)
	}

	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email '%s', got '%s'", user.Email, retrievedUser.Email)
	}

	// Test getting non-existent session
	_, err = store.GetSession(ctx, "nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}

	// Test deleting session
	err = store.DeleteSession(ctx, sessionID)
	if err != nil {
		t.Errorf("Failed to delete session: %v", err)
	}

	// Test that deleted session is gone
	_, err = store.GetSession(ctx, sessionID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound after deletion, got %v", err)
	}
}

func TestMemorySessionStoreWithTimeout(t *testing.T) {
	// Create store with very short timeout for testing
	timeout := 100 * time.Millisecond
	store := NewMemorySessionStoreWithTimeout(timeout)
	ctx := context.Background()

	user := &AuthUser{
		ID:       "test456",
		Username: "testuser2",
		Email:    "test2@example.com",
		Roles:    []string{"user"},
	}

	// Create session
	sessionID, err := store.CreateSession(ctx, user)
	if err != nil {
		t.Errorf("Failed to create session: %v", err)
	}

	// Session should exist immediately
	_, err = store.GetSession(ctx, sessionID)
	if err != nil {
		t.Errorf("Session should exist immediately after creation: %v", err)
	}

	// Wait for session to expire
	time.Sleep(timeout + 50*time.Millisecond)

	// Session should be expired
	_, err = store.GetSession(ctx, sessionID)
	if err != ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired, got %v", err)
	}
}

func TestMemorySessionStoreCleanup(t *testing.T) {
	store := NewMemorySessionStoreWithTimeout(50 * time.Millisecond)
	ctx := context.Background()

	user := &AuthUser{
		ID:       "test789",
		Username: "testuser3",
		Email:    "test3@example.com",
		Roles:    []string{"user"},
	}

	// Create multiple sessions
	sessionID1, _ := store.CreateSession(ctx, user)
	sessionID2, _ := store.CreateSession(ctx, user)

	initialCount := store.GetSessionCount()
	if initialCount != 2 {
		t.Errorf("Expected 2 sessions, got %d", initialCount)
	}

	// Wait for sessions to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	store.CleanExpiredSessions(ctx)

	// Check that sessions were cleaned up
	finalCount := store.GetSessionCount()
	if finalCount != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", finalCount)
	}

	// Verify sessions are actually gone
	_, err1 := store.GetSession(ctx, sessionID1)
	_, err2 := store.GetSession(ctx, sessionID2)

	if err1 != ErrSessionNotFound || err2 != ErrSessionNotFound {
		t.Error("Expected both sessions to be not found after cleanup")
	}
}
