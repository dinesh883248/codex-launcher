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
	if err != nil {
		return err
	}

	// output_lines table - each line of codex output stored as a row
	_, err = s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS output_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id INTEGER NOT NULL,
			line_num INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (request_id) REFERENCES requests(id)
		)`,
	)
	if err != nil {
		return err
	}

	// index for fast lookup by request_id
	_, _ = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_output_lines_request ON output_lines(request_id, line_num)`)

	return nil
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
		`SELECT id, prompt, status, response, created_at
		FROM requests
		ORDER BY id DESC
		LIMIT ? OFFSET ?`,
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
	return items, total, rows.Err()
}

func (s *Store) GetProcessingRequest(ctx context.Context) (Request, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, prompt, status, response, created_at
		FROM requests
		WHERE status = ?
		ORDER BY id DESC
		LIMIT 1`,
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

func (s *Store) GetRequest(ctx context.Context, id int64) (Request, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, prompt, status, response, created_at FROM requests WHERE id = ?`,
		id,
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

// AddOutputLine inserts a single line of output for a request
func (s *Store) AddOutputLine(ctx context.Context, requestID int64, lineNum int, content string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO output_lines (request_id, line_num, content, created_at) VALUES (?, ?, ?, ?)",
		requestID, lineNum, content, now,
	)
	return err
}

// GetOutputLines returns output lines for a request with pagination (returns last N lines before offset)
func (s *Store) GetOutputLines(ctx context.Context, requestID int64, limit, offset int) ([]OutputLine, int, error) {
	// get total count
	row := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM output_lines WHERE request_id = ?", requestID)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	// get lines ordered by line_num descending (newest first), with pagination
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, request_id, line_num, content, created_at
		FROM output_lines
		WHERE request_id = ?
		ORDER BY line_num DESC
		LIMIT ? OFFSET ?`,
		requestID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lines []OutputLine
	for rows.Next() {
		var line OutputLine
		if err := rows.Scan(&line.ID, &line.RequestID, &line.LineNum, &line.Content, &line.CreatedAt); err != nil {
			return nil, 0, err
		}
		lines = append(lines, line)
	}
	return lines, total, rows.Err()
}

// GetNextLineNum returns the next line number for a request
func (s *Store) GetNextLineNum(ctx context.Context, requestID int64) (int, error) {
	row := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(line_num), 0) + 1 FROM output_lines WHERE request_id = ?", requestID)
	var next int
	err := row.Scan(&next)
	return next, err
}
