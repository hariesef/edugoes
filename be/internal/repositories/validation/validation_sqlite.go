package validation

import (
	"context"
	"database/sql"
	"errors"
	"time"

	vrepo "github.com/quipper/poc/lti/be/pkg/repositories/validation"
	_ "modernc.org/sqlite"
)

// SQLiteRepo is a separate SQLite-backed repo for validation concerns (JTIs, OIDC state).
type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(dsn string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Pragmas safe for simple single-process usage
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		return nil, err
	}
	return &SQLiteRepo{db: db}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS client_assertion_jtis (
    jti TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_jtis_expires_at ON client_assertion_jtis(expires_at);

CREATE TABLE IF NOT EXISTS oidc_states (
    state TEXT PRIMARY KEY,
    client_id TEXT,
    target_link_uri TEXT,
    resource_link_id TEXT,
    context_id TEXT,
    expires_at TIMESTAMP NOT NULL,
    used INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_states_expires_at ON oidc_states(expires_at);
`)
	if err != nil {
		return err
	}
	// Best-effort: add columns for existing databases; ignore error if exists
	_, _ = db.Exec(`ALTER TABLE oidc_states ADD COLUMN context_id TEXT`)
	_, _ = db.Exec(`ALTER TABLE oidc_states ADD COLUMN resource_link_id TEXT`)
	return nil
}

func (r *SQLiteRepo) Disconnect() { _ = r.db.Close() }

// Ensure interface compliance
var _ vrepo.Repository = (*SQLiteRepo)(nil)

func (r *SQLiteRepo) TryUseClientAssertionJTI(ctx context.Context, jti string, clientID string, exp time.Time) (bool, error) {
	if jti == "" {
		return false, errors.New("empty jti")
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	// Cleanup expired quickly (best-effort)
	if _, _ = tx.ExecContext(ctx, "DELETE FROM client_assertion_jtis WHERE expires_at < CURRENT_TIMESTAMP"); false {
	}

	// Insert if not exists
	_, err = tx.ExecContext(ctx, `INSERT INTO client_assertion_jtis (jti, client_id, expires_at) VALUES (?, ?, ?)`, jti, clientID, exp.UTC())
	if err != nil {
		// Likely unique constraint -> replay
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *SQLiteRepo) CreateOIDCState(ctx context.Context, state string, clientID string, targetLinkURI string, contextID string, resourceLinkID string, exp time.Time) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Cleanup expired
	if _, _ = tx.ExecContext(ctx, "DELETE FROM oidc_states WHERE expires_at < CURRENT_TIMESTAMP OR used = 1"); false {
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO oidc_states (state, client_id, target_link_uri, resource_link_id, context_id, expires_at) VALUES (?, ?, ?, ?, ?, ?)`, state, clientID, targetLinkURI, resourceLinkID, contextID, exp.UTC())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepo) ConsumeOIDCState(ctx context.Context, state string) (clientID string, targetLinkURI string, resourceLinkID string, contextID string, ok bool, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", "", "", "", false, err
	}
	defer func() { _ = tx.Rollback() }()

	// Load
	row := tx.QueryRowContext(ctx, `SELECT client_id, target_link_uri, resource_link_id, context_id, expires_at, used FROM oidc_states WHERE state = ?`, state)
	var exp time.Time
	var used int
	if err := row.Scan(&clientID, &targetLinkURI, &resourceLinkID, &contextID, &exp, &used); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", "", false, nil
		}
		return "", "", "", "", false, err
	}
	if used == 1 || time.Now().After(exp) {
		return "", "", "", "", false, nil
	}
	// Mark used
	if _, err := tx.ExecContext(ctx, `UPDATE oidc_states SET used = 1 WHERE state = ?`, state); err != nil {
		return "", "", "", "", false, err
	}
	if err := tx.Commit(); err != nil {
		return "", "", "", "", false, err
	}
	return clientID, targetLinkURI, resourceLinkID, contextID, true, nil
}
