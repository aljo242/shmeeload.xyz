package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (works with CGO disabled)
)

// chatHistoryLimit is how many recent messages a joining client is replayed.
const chatHistoryLimit = 100

// chatStore persists chat messages per room in SQLite. The pure-Go driver keeps
// the CGO-free build, and the DB lives on the writable /data volume so messages
// survive restarts.
type chatStore struct {
	db *sql.DB
}

func newChatStore(path string) (*chatStore, error) {
	// WAL + a busy timeout let the single writer and history readers coexist.
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening chat db: %w", err)
	}
	// One connection serializes access and avoids SQLite "database is locked".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		room       TEXT NOT NULL,
		body       BLOB NOT NULL,
		created_at INTEGER NOT NULL
	)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating messages table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room, id)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating index: %w", err)
	}
	return &chatStore{db: db}, nil
}

// save records one message for a room.
func (s *chatStore) save(room string, body []byte) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (room, body, created_at) VALUES (?, ?, ?)`,
		room, body, time.Now().Unix())
	return err
}

// recent returns up to limit of a room's messages, oldest first.
func (s *chatStore) recent(room string, limit int) ([][]byte, error) {
	rows, err := s.db.Query(
		`SELECT body FROM messages WHERE room = ? ORDER BY id DESC LIMIT ?`, room, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][]byte
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Query was newest-first; reverse to chronological for replay.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// purgeOlderThan deletes messages older than d and returns how many were removed.
func (s *chatStore) purgeOlderThan(d time.Duration) (int64, error) {
	res, err := s.db.Exec(
		`DELETE FROM messages WHERE created_at < ?`, time.Now().Add(-d).Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *chatStore) close() error { return s.db.Close() }
