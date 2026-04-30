package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Create directory if needed
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database tables
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		metadata TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		tool_calls TEXT,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateSession creates a new session
func (s *SQLiteStore) CreateSession(ctx context.Context, session *Session) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO sessions (id, created_at, updated_at, provider, model, metadata)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		session.ID,
		session.CreatedAt,
		session.UpdatedAt,
		session.Provider,
		session.Model,
		string(metadataJSON),
	)

	return err
}

// GetSession retrieves a session by ID
func (s *SQLiteStore) GetSession(ctx context.Context, id string) (*Session, error) {
	query := `
	SELECT id, created_at, updated_at, provider, model, metadata
	FROM sessions WHERE id = ?
	`

	session := &Session{}
	var metadataJSON []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.Provider,
		&session.Model,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, err
	}

	if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
		return nil, err
	}

	// Load messages
	messages, err := s.GetMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	session.Messages = messages

	return session, nil
}

// UpdateSession updates an existing session
func (s *SQLiteStore) UpdateSession(ctx context.Context, session *Session) error {
	session.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return err
	}

	query := `
	UPDATE sessions SET updated_at = ?, provider = ?, model = ?, metadata = ?
	WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		session.UpdatedAt,
		session.Provider,
		session.Model,
		string(metadataJSON),
		session.ID,
	)

	return err
}

// DeleteSession removes a session
func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

// ListSessions returns all sessions
func (s *SQLiteStore) ListSessions(ctx context.Context, limit, offset int) ([]Session, error) {
	query := `
	SELECT id, created_at, updated_at, provider, model, metadata
	FROM sessions ORDER BY updated_at DESC LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var metadataJSON []byte

		if err := rows.Scan(
			&session.ID,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.Provider,
			&session.Model,
			&metadataJSON,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// AddMessage appends a message to a session
func (s *SQLiteStore) AddMessage(ctx context.Context, sessionID string, msg *Message) error {
	toolCallsJSON, err := json.Marshal(msg.ToolCalls)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO messages (id, session_id, role, content, timestamp, tool_calls)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		msg.ID,
		sessionID,
		msg.Role,
		msg.Content,
		msg.Timestamp,
		string(toolCallsJSON),
	)

	// Update session timestamp
	if err == nil {
		_, err = s.db.ExecContext(ctx,
			"UPDATE sessions SET updated_at = ? WHERE id = ?",
			time.Now(), sessionID,
		)
	}

	return err
}

// GetMessages returns all messages for a session
func (s *SQLiteStore) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	query := `
	SELECT id, role, content, timestamp, tool_calls
	FROM messages WHERE session_id = ? ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var toolCallsJSON []byte

		if err := rows.Scan(
			&msg.ID,
			&msg.Role,
			&msg.Content,
			&msg.Timestamp,
			&toolCallsJSON,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(toolCallsJSON, &msg.ToolCalls); err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
