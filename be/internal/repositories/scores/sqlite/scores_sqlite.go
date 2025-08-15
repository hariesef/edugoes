package sqlite

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"

	sc "github.com/quipper/poc/lti/be/pkg/repositories/scores"
)

type SQLiteRepo struct {
	db *sql.DB
}

// CreateLineItemMapping creates a one-to-one mapping between lineItemID and resourceLinkID.
// Both columns are UNIQUE to ensure one-to-one mapping.
func (r *SQLiteRepo) CreateLineItemMapping(ctx context.Context, lineItemID int64, resourceLinkID string) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO line_item_mappings (line_item_id, resource_link_id)
        VALUES (?, ?)
    `, lineItemID, resourceLinkID)
	return err
}

// GetLineItemIDByResourceLinkID returns the line_item_id for a given resource_link_id.
// Returns 0 and nil error if not found.
func (r *SQLiteRepo) GetLineItemIDByResourceLinkID(ctx context.Context, resourceLinkID string) (int64, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT line_item_id FROM line_item_mappings WHERE resource_link_id = ?
    `, resourceLinkID)
	var id sql.NullInt64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if id.Valid {
		return id.Int64, nil
	}
	return 0, nil
}

func NewSQLiteRepo(path string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteRepo{db: db}, nil
}

func (r *SQLiteRepo) Disconnect() {
	_ = r.db.Close()
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS line_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			context_id TEXT NOT NULL,
			label TEXT NOT NULL,
			resource_id TEXT,
			resource_link_id TEXT,
			tag TEXT,
			score_maximum REAL NOT NULL,
			start_at TIMESTAMP,
			end_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS line_item_mappings (
			line_item_id INTEGER NOT NULL UNIQUE,
			resource_link_id TEXT NOT NULL UNIQUE
		);
		CREATE TABLE IF NOT EXISTS results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			line_item_id INTEGER NOT NULL,
			context_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			result_score REAL,
			result_maximum REAL,
			comment TEXT,
			timestamp TIMESTAMP NOT NULL,
			activity_progress TEXT,
			grading_progress TEXT,
			UNIQUE(line_item_id, context_id, user_id)
		);
	`)
	if err != nil {
		return err
	}
	// Lightweight migration for older DBs missing new columns
	if err := migrateResultsColumns(db); err != nil {
		return err
	}
	return nil
}

func migrateResultsColumns(db *sql.DB) error {
	// add columns if they don't exist
	needActivity := true
	needGrading := true
	rows, err := db.Query(`PRAGMA table_info(results)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		switch name {
		case "activity_progress":
			needActivity = false
		case "grading_progress":
			needGrading = false
		}
	}
	if needActivity {
		if _, err := db.Exec(`ALTER TABLE results ADD COLUMN activity_progress TEXT`); err != nil {
			return err
		}
	}
	if needGrading {
		if _, err := db.Exec(`ALTER TABLE results ADD COLUMN grading_progress TEXT`); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLiteRepo) CreateLineItem(ctx context.Context, li *sc.LineItem) (int64, error) {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO line_items (context_id, label, resource_id, resource_link_id, tag, score_maximum, start_at, end_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, li.ContextID, li.Label, li.ResourceID, li.ResourceLinkID, li.Tag, li.ScoreMaximum, nullableTime(li.StartAt), nullableTime(li.EndAt), now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	li.ID = id
	li.CreatedAt = now
	li.UpdatedAt = now
	return id, nil
}

func (r *SQLiteRepo) ListLineItems(ctx context.Context, contextID string) ([]*sc.LineItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, context_id, label, resource_id, resource_link_id, tag, score_maximum, start_at, end_at, created_at, updated_at
		FROM line_items WHERE context_id = ? ORDER BY id ASC`, contextID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*sc.LineItem
	for rows.Next() {
		var li sc.LineItem
		var start, end, created, updated sql.NullTime
		if err := rows.Scan(&li.ID, &li.ContextID, &li.Label, &li.ResourceID, &li.ResourceLinkID, &li.Tag, &li.ScoreMaximum, &start, &end, &created, &updated); err != nil {
			return nil, err
		}
		if start.Valid {
			li.StartAt = &start.Time
		}
		if end.Valid {
			li.EndAt = &end.Time
		}
		if created.Valid {
			li.CreatedAt = created.Time
		}
		if updated.Valid {
			li.UpdatedAt = updated.Time
		}
		out = append(out, &li)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) GetLineItem(ctx context.Context, id int64, contextID string) (*sc.LineItem, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, context_id, label, resource_id, resource_link_id, tag, score_maximum, start_at, end_at, created_at, updated_at
		FROM line_items WHERE id = ? AND context_id = ?`, id, contextID)
	var li sc.LineItem
	var start, end, created, updated sql.NullTime
	if err := row.Scan(&li.ID, &li.ContextID, &li.Label, &li.ResourceID, &li.ResourceLinkID, &li.Tag, &li.ScoreMaximum, &start, &end, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if start.Valid {
		li.StartAt = &start.Time
	}
	if end.Valid {
		li.EndAt = &end.Time
	}
	if created.Valid {
		li.CreatedAt = created.Time
	}
	if updated.Valid {
		li.UpdatedAt = updated.Time
	}
	return &li, nil
}

func (r *SQLiteRepo) UpdateLineItem(ctx context.Context, li *sc.LineItem) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE line_items SET label = ?, resource_id = ?, resource_link_id = ?, tag = ?, score_maximum = ?, start_at = ?, end_at = ?, updated_at = ?
		WHERE id = ? AND context_id = ?
	`, li.Label, li.ResourceID, li.ResourceLinkID, li.Tag, li.ScoreMaximum, nullableTime(li.StartAt), nullableTime(li.EndAt), now, li.ID, li.ContextID)
	if err == nil {
		li.UpdatedAt = now
	}
	return err
}

func (r *SQLiteRepo) DeleteLineItem(ctx context.Context, id int64, contextID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM line_items WHERE id = ? AND context_id = ?`, id, contextID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepo) UpsertResultFromScore(ctx context.Context, lineItemID int64, contextID string, s *sc.Score) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO results (line_item_id, context_id, user_id, result_score, result_maximum, comment, timestamp, activity_progress, grading_progress)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(line_item_id, context_id, user_id)
		DO UPDATE SET result_score = excluded.result_score, result_maximum = excluded.result_maximum, comment = excluded.comment, timestamp = excluded.timestamp, activity_progress = excluded.activity_progress, grading_progress = excluded.grading_progress
	`, lineItemID, contextID, s.UserID, nullableFloat(s.ScoreGiven), nullableFloat(s.ScoreMaximum), s.Comment, s.Timestamp.UTC(), s.ActivityProgress, s.GradingProgress)
	return err
}

func (r *SQLiteRepo) ListResultsByLineItem(ctx context.Context, lineItemID int64, contextID string) ([]*sc.Result, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, result_score, result_maximum, comment, timestamp, activity_progress, grading_progress
		FROM results WHERE line_item_id = ? AND context_id = ? ORDER BY user_id ASC`, lineItemID, contextID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*sc.Result
	for rows.Next() {
		var rscore sc.Result
		var score, max sql.NullFloat64
		var ts time.Time
		var comment sql.NullString
		var act sql.NullString
		var grd sql.NullString
		if err := rows.Scan(&rscore.UserID, &score, &max, &comment, &ts, &act, &grd); err != nil {
			return nil, err
		}
		if score.Valid {
			v := score.Float64
			rscore.ResultScore = &v
		}
		if max.Valid {
			v := max.Float64
			rscore.ResultMaximum = &v
		}
		if comment.Valid {
			rscore.Comment = comment.String
		}
		rscore.Timestamp = ts
		if act.Valid {
			rscore.ActivityProgress = act.String
		}
		if grd.Valid {
			rscore.GradingProgress = grd.String
		}
		out = append(out, &rscore)
	}
	return out, rows.Err()
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func nullableFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}
