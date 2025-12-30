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
	if err := s.ensureColumn(ctx, "cast_start", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "cast_end", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "cast_path", "TEXT"); err != nil {
		return err
	}
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
		`SELECT id, prompt, status, response, created_at, cast_start, cast_end, cast_path
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
		var castStart sql.NullFloat64
		var castEnd sql.NullFloat64
		var castPath sql.NullString
		if err := rows.Scan(
			&req.ID,
			&req.Prompt,
			&req.Status,
			&req.Response,
			&req.CreatedAt,
			&castStart,
			&castEnd,
			&castPath,
		); err != nil {
			return nil, 0, err
		}
		if castStart.Valid {
			req.CastStart = &castStart.Float64
		}
		if castEnd.Valid {
			req.CastEnd = &castEnd.Float64
		}
		if castPath.Valid {
			req.CastPath = castPath.String
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
		`SELECT id, prompt, status, response, created_at, cast_start, cast_end, cast_path
		FROM requests
		WHERE status = ?
		ORDER BY id DESC
		LIMIT 1`,
		"processing",
	)
	var req Request
	var castStart sql.NullFloat64
	var castEnd sql.NullFloat64
	var castPath sql.NullString
	if err := row.Scan(
		&req.ID,
		&req.Prompt,
		&req.Status,
		&req.Response,
		&req.CreatedAt,
		&castStart,
		&castEnd,
		&castPath,
	); err != nil {
		if err == sql.ErrNoRows {
			return Request{}, false, nil
		}
		return Request{}, false, err
	}
	if castStart.Valid {
		req.CastStart = &castStart.Float64
	}
	if castEnd.Valid {
		req.CastEnd = &castEnd.Float64
	}
	if castPath.Valid {
		req.CastPath = castPath.String
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

func (s *Store) UpdateRequestCastStart(ctx context.Context, id int64, start float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE requests SET cast_start = ?, updated_at = ? WHERE id = ?",
		start,
		now,
		id,
	)
	return err
}

func (s *Store) UpdateRequestFinal(ctx context.Context, id int64, status, response string, castEnd *float64, castPath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var endVal sql.NullFloat64
	if castEnd != nil {
		endVal = sql.NullFloat64{Float64: *castEnd, Valid: true}
	}
	var pathVal sql.NullString
	if castPath != "" {
		pathVal = sql.NullString{String: castPath, Valid: true}
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE requests
		SET status = ?, response = ?, cast_end = ?, cast_path = ?, updated_at = ?
		WHERE id = ?`,
		status,
		response,
		endVal,
		pathVal,
		now,
		id,
	)
	return err
}

func (s *Store) GetRequest(ctx context.Context, id int64) (Request, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, prompt, status, response, created_at, cast_start, cast_end, cast_path
		FROM requests
		WHERE id = ?`,
		id,
	)
	var req Request
	var castStart sql.NullFloat64
	var castEnd sql.NullFloat64
	var castPath sql.NullString
	if err := row.Scan(
		&req.ID,
		&req.Prompt,
		&req.Status,
		&req.Response,
		&req.CreatedAt,
		&castStart,
		&castEnd,
		&castPath,
	); err != nil {
		if err == sql.ErrNoRows {
			return Request{}, false, nil
		}
		return Request{}, false, err
	}
	if castStart.Valid {
		req.CastStart = &castStart.Float64
	}
	if castEnd.Valid {
		req.CastEnd = &castEnd.Float64
	}
	if castPath.Valid {
		req.CastPath = castPath.String
	}
	return req, true, nil
}

func (s *Store) ensureColumn(ctx context.Context, name, columnType string) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info(requests)")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var colName string
		var colType string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if colName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		"ALTER TABLE requests ADD COLUMN "+name+" "+columnType,
	)
	return err
}
