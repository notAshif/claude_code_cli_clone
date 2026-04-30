package agent

import (
	"context"
	"time"

	"github.com/asif/gocode-agent/internal/storage"
)

// Session manages conversation state and persistence
type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Provider  string
	Model     string
	Messages  []Message
	Metadata  SessionMetadata
	store     storage.Store
}

// SessionMetadata holds session metadata
type SessionMetadata struct {
	WorkingDir  string
	TokenUsage  TokenUsage
	ToolCount   int
	LastCommand string
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// NewSession creates a new session
func NewSession(provider, model, workingDir string) *Session {
	now := time.Now()
	return &Session{
		ID:        generateID(),
		CreatedAt: now,
		UpdatedAt: now,
		Provider:  provider,
		Model:     model,
		Messages:  make([]Message, 0),
		Metadata: SessionMetadata{
			WorkingDir: workingDir,
		},
	}
}

// LoadSession loads a session from storage
func LoadSession(ctx context.Context, store storage.Store, id string) (*Session, error) {
	stored, err := store.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        stored.ID,
		CreatedAt: stored.CreatedAt,
		UpdatedAt: stored.UpdatedAt,
		Provider:  stored.Provider,
		Model:     stored.Model,
		Messages:  make([]Message, len(stored.Messages)),
		Metadata: SessionMetadata{
			WorkingDir: stored.Metadata.WorkingDir,
			TokenUsage: TokenUsage{
				InputTokens:  stored.Metadata.TokenUsage,
			},
		},
		store: store,
	}

	// Convert storage messages to agent messages
	for i, msg := range stored.Messages {
		session.Messages[i] = Message{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
	}

	return session, nil
}

// AddMessage appends a message to the session
func (s *Session) AddMessage(msg Message) {
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

// Save persists the session
func (s *Session) Save(ctx context.Context) error {
	if s.store == nil {
		return nil // No store configured
	}

	if err := s.store.UpdateSession(ctx, s.toStorageSession()); err != nil {
		return err
	}

	// Save messages
	for _, msg := range s.Messages {
		if err := s.store.AddMessage(ctx, s.ID, s.messageToStorage(msg)); err != nil {
			return err
		}
	}

	return nil
}

// CreateInStore creates the session in storage
func (s *Session) CreateInStore(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.CreateSession(ctx, s.toStorageSession())
}

// Delete removes the session from storage
func (s *Session) Delete(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.DeleteSession(ctx, s.ID)
}

// GetMessages returns all messages
func (s *Session) GetMessages() []Message {
	return s.Messages
}

// GetMessageCount returns the number of messages
func (s *Session) GetMessageCount() int {
	return len(s.Messages)
}

// GetLastMessage returns the most recent message
func (s *Session) GetLastMessage() *Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// ClearMessages removes all messages
func (s *Session) ClearMessages() {
	s.Messages = make([]Message, 0)
	s.UpdatedAt = time.Now()
}

// UpdateTokenUsage updates token usage statistics
func (s *Session) UpdateTokenUsage(input, output int) {
	s.Metadata.TokenUsage.InputTokens += input
	s.Metadata.TokenUsage.OutputTokens += output
	s.Metadata.TokenUsage.TotalTokens += input + output
}

// IncrementToolCount increments the tool call counter
func (s *Session) IncrementToolCount() {
	s.Metadata.ToolCount++
}

// SetLastCommand records the last command
func (s *Session) SetLastCommand(cmd string) {
	s.Metadata.LastCommand = cmd
	s.UpdatedAt = time.Now()
}

func (s *Session) toStorageSession() *storage.Session {
	return &storage.Session{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Provider:  s.Provider,
		Model:     s.Model,
		Metadata: storage.Metadata{
			WorkingDir:  s.Metadata.WorkingDir,
			TokenUsage:  s.Metadata.TokenUsage.TotalTokens,
			ToolCount:   s.Metadata.ToolCount,
			LastCommand: s.Metadata.LastCommand,
		},
	}
}

func (s *Session) messageToStorage(msg Message) *storage.Message {
	return &storage.Message{
		ID:        msg.ID,
		Role:      msg.Role,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}
}
