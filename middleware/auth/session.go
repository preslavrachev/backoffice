package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	// ErrSessionNotFound is returned when a session cannot be found
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session has expired
	ErrSessionExpired = errors.New("session expired")
)

// SessionData holds session information
type SessionData struct {
	User      *AuthUser
	CreatedAt time.Time
	ExpiresAt time.Time
}

// IsExpired checks if the session has expired
func (s *SessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// MemorySessionStore implements SessionStore using in-memory storage
// This is suitable for single-instance applications or development
// For production multi-instance deployments, consider Redis or database-backed storage
type MemorySessionStore struct {
	sessions map[string]*SessionData
	mutex    sync.RWMutex

	// SessionTimeout defines how long sessions last (default: 24 hours)
	SessionTimeout time.Duration
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	store := &MemorySessionStore{
		sessions:       make(map[string]*SessionData),
		SessionTimeout: 24 * time.Hour, // Default 24 hour sessions
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// NewMemorySessionStoreWithTimeout creates a new in-memory session store with custom timeout
func NewMemorySessionStoreWithTimeout(timeout time.Duration) *MemorySessionStore {
	store := &MemorySessionStore{
		sessions:       make(map[string]*SessionData),
		SessionTimeout: timeout,
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// GetSession retrieves a user session by session ID
func (m *MemorySessionStore) GetSession(ctx context.Context, sessionID string) (*AuthUser, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessionData, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	if sessionData.IsExpired() {
		// Clean up expired session
		go func() {
			m.mutex.Lock()
			delete(m.sessions, sessionID)
			m.mutex.Unlock()
		}()
		return nil, ErrSessionExpired
	}

	return sessionData.User, nil
}

// CreateSession creates a new session for the user and returns the session ID
func (m *MemorySessionStore) CreateSession(ctx context.Context, user *AuthUser) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	sessionData := &SessionData{
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.SessionTimeout),
	}

	m.mutex.Lock()
	m.sessions[sessionID] = sessionData
	m.mutex.Unlock()

	return sessionID, nil
}

// DeleteSession removes a session by session ID
func (m *MemorySessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// CleanExpiredSessions removes expired sessions
func (m *MemorySessionStore) CleanExpiredSessions(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, sessionData := range m.sessions {
		if now.After(sessionData.ExpiresAt) {
			delete(m.sessions, sessionID)
		}
	}

	return nil
}

// cleanupLoop runs periodically to clean up expired sessions
func (m *MemorySessionStore) cleanupLoop() {
	ticker := time.NewTicker(time.Hour) // Clean up every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.CleanExpiredSessions(context.Background())
		}
	}
}

// GetSessionCount returns the current number of active sessions (for debugging/monitoring)
func (m *MemorySessionStore) GetSessionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.sessions)
}

// generateSessionID creates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
