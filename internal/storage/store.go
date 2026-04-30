package storage

import (
	"context"
	"time"
)

// Session represents a stored conversation session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Metadata  Metadata  `json:"metadata"`
}

// Message represents a conversation message
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	ToolCalls []string  `json:"tool_calls,omitempty"`
}

// Metadata holds session metadata
type Metadata struct {
	WorkingDir   string `json:"working_dir"`
	TokenUsage   int    `json:"token_usage"`
	ToolCount    int    `json:"tool_count"`
	LastCommand  string `json:"last_command"`
}

// Store interface for session persistence
type Store interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, session *Session) error

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, id string) (*Session, error)

	// UpdateSession updates an existing session
	UpdateSession(ctx context.Context, session *Session) error

	// DeleteSession removes a session
	DeleteSession(ctx context.Context, id string) error

	// ListSessions returns all sessions
	ListSessions(ctx context.Context, limit int, offset int) ([]Session, error)

	// AddMessage appends a message to a session
	AddMessage(ctx context.Context, sessionID string, msg *Message) error

	// GetMessages returns all messages for a session
	GetMessages(ctx context.Context, sessionID string) ([]Message, error)

	// Close closes the store
	Close() error
}

// StoreConfig holds storage configuration
type StoreConfig struct {
	Path string
}

// DefaultStoreConfig returns default configuration
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		Path: ".agent/sessions.db",
	}
}
