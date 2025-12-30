package api

import (
	"context"
	"database/sql"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Init(ctx context.Context) error {
	_, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt TEXT NOT NULL,
			status TEXT NOT NULL,
			response TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	)
	return err
}

func (s *Store) CreateRequest(ctx context.Context, prompt string) (Request, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(
		ctx,
		"INSERT INTO requests (prompt, status, response, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		prompt,
		"pending",
		"",
		now,
		now,
	)
	if err != nil {
		return Request{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Request{}, err
	}
	return Request{
		ID:        id,
		Prompt:    prompt,
		Status:    "pending",
		Response:  "",
		CreatedAt: now,
	}, nil
}

func (s *Store) ListRequests(ctx context.Context, offset, limit int) ([]Request, int, error) {
	row := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM requests")
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT id, prompt, status, response, created_at FROM requests ORDER BY id DESC LIMIT ? OFFSET ?",
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Request{}
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.Prompt, &req.Status, &req.Response, &req.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, req)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) GetProcessingRequest(ctx context.Context) (Request, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		"SELECT id, prompt, status, response, created_at FROM requests WHERE status = ? ORDER BY id DESC LIMIT 1",
		"processing",
	)
	var req Request
	if err := row.Scan(&req.ID, &req.Prompt, &req.Status, &req.Response, &req.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Request{}, false, nil
		}
		return Request{}, false, err
	}
	return req, true, nil
}

func (s *Store) ClaimNextPending(ctx context.Context) (Request, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Request{}, false, err
	}
	row := tx.QueryRowContext(
		ctx,
		"SELECT id, prompt FROM requests WHERE status = ? ORDER BY id LIMIT 1",
		"pending",
	)
	var req Request
	if err := row.Scan(&req.ID, &req.Prompt); err != nil {
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return Request{}, false, nil
		}
		_ = tx.Rollback()
		return Request{}, false, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(
		ctx,
		"UPDATE requests SET status = ?, updated_at = ? WHERE id = ?",
		"processing",
		now,
		req.ID,
	); err != nil {
		_ = tx.Rollback()
		return Request{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Request{}, false, err
	}
	return req, true, nil
}

func (s *Store) UpdateRequest(ctx context.Context, id int64, status, response string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE requests SET status = ?, response = ?, updated_at = ? WHERE id = ?",
		status,
		response,
		now,
		id,
	)
	return err
}
