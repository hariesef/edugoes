package sqlite

import (
	"database/sql"
	"context"
	"time"

	_ "modernc.org/sqlite"
	"sync"

	repoIface "github.com/quipper/poc/lti/be/pkg/repositories/lti"
)

type SQLiteRepo struct {
	db *sql.DB
	wg *sync.WaitGroup
}

// CreateOIDCState stores a state/nonce with metadata and expiry.
func (r *SQLiteRepo) CreateOIDCState(ctx context.Context, state string, nonce string, clientID string, targetLinkURI string, exp time.Time) error {
    _, err := r.db.ExecContext(ctx, `INSERT INTO oidc_states (state, nonce, client_id, target_link_uri, expires_at) VALUES (?, ?, ?, ?, ?)`, state, nonce, clientID, targetLinkURI, exp.UTC())
    if err != nil {
        return err
    }
    return nil
}

// ConsumeOIDCState atomically marks a state as used and returns data if valid.
func (r *SQLiteRepo) ConsumeOIDCState(ctx context.Context, state string) (nonce string, clientID string, targetLinkURI string, ok bool, err error) {
    tx, err2 := r.db.BeginTx(ctx, nil)
    if err2 != nil {
        err = err2
        return
    }
    defer func() { _ = tx.Rollback() }()

    var expiresAt time.Time
    var used int
    row := tx.QueryRowContext(ctx, `SELECT nonce, client_id, target_link_uri, expires_at, used FROM oidc_states WHERE state = ?`, state)
    if err2 = row.Scan(&nonce, &clientID, &targetLinkURI, &expiresAt, &used); err2 != nil {
        if err2 == sql.ErrNoRows {
            ok = false
            err = nil
            return
        }
        err = err2
        return
    }
    if used != 0 || time.Now().After(expiresAt) {
        ok = false
        err = nil
        return
    }
    if _, err2 = tx.ExecContext(ctx, `UPDATE oidc_states SET used = 1 WHERE state = ?`, state); err2 != nil {
        err = err2
        return
    }
    if err2 = tx.Commit(); err2 != nil {
        err = err2
        return
    }
    ok = true
    return
}

// CreateDeepLinkSelection inserts a new selection row and returns its ID.
func (r *SQLiteRepo) CreateDeepLinkSelection(ctx context.Context, sel *repoIface.DeepLinkSelection) (int64, error) {
    now := time.Now().UTC()
    res, err := r.db.ExecContext(ctx, `
        INSERT INTO deeplink_selections (client_id, tool_name, url, content_item_json, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, sel.ClientID, sel.ToolName, sel.URL, sel.ContentItemJSON, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	sel.ID = id
	sel.CreatedAt = now
	return id, nil
}

// ListDeepLinkSelections returns all selections ordered by newest first.
func (r *SQLiteRepo) ListDeepLinkSelections(ctx context.Context) ([]*repoIface.DeepLinkSelection, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, client_id, tool_name, url, content_item_json, created_at
        FROM deeplink_selections ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*repoIface.DeepLinkSelection
	for rows.Next() {
		var s repoIface.DeepLinkSelection
		var created time.Time
		        if err := rows.Scan(&s.ID, &s.ClientID, &s.ToolName, &s.URL, &s.ContentItemJSON, &created); err != nil {
            return nil, err
        }
		s.CreatedAt = created
		out = append(out, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDeepLinkSelection returns a selection by ID.
func (r *SQLiteRepo) GetDeepLinkSelection(ctx context.Context, id int64) (*repoIface.DeepLinkSelection, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT id, client_id, tool_name, url, content_item_json, created_at
        FROM deeplink_selections WHERE id = ?`, id)
    var s repoIface.DeepLinkSelection
    var created time.Time
    if err := row.Scan(&s.ID, &s.ClientID, &s.ToolName, &s.URL, &s.ContentItemJSON, &created); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    s.CreatedAt = created
    return &s, nil
}

// DeleteDeepLinkSelection deletes a selection by ID.
func (r *SQLiteRepo) DeleteDeepLinkSelection(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM deeplink_selections WHERE id = ?`, id)
	return err
}

func NewSQLiteRepo(path string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		return nil, err
	}
    // Best-effort migration: add target_link_url if column not yet present
    _, _ = db.Exec(`ALTER TABLE tools ADD COLUMN target_link_url TEXT`)
	return &SQLiteRepo{db: db, wg: &sync.WaitGroup{}}, nil
}

func initSchema(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS tools (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            client_id TEXT NOT NULL UNIQUE,
            auth_url TEXT,
            target_link_url TEXT,
            token_url TEXT,
            key_set_url TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS oidc_states (
            state TEXT PRIMARY KEY,
            nonce TEXT NOT NULL,
            client_id TEXT,
            target_link_uri TEXT,
            expires_at TIMESTAMP NOT NULL,
            used INTEGER NOT NULL DEFAULT 0,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS client_assertion_jtis (
            jti TEXT PRIMARY KEY,
            client_id TEXT NOT NULL,
            expires_at TIMESTAMP NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
		CREATE TABLE IF NOT EXISTS deeplink_selections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT,
			tool_name TEXT,
			url TEXT,
			content_item_json TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (r *SQLiteRepo) Health() error {
	return r.db.Ping()
}

// Disconnect waits for ongoing tasks and closes DB
func (r *SQLiteRepo) Disconnect() {
	if r.wg != nil {
		r.wg.Wait()
	}
	_ = r.db.Close()
}

// RegisterTool inserts a new tool and returns its ID
func (r *SQLiteRepo) RegisterTool(ctx context.Context, t *repoIface.Tool) (int64, error) {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
        INSERT INTO tools (name, client_id, auth_url, target_link_url, token_url, key_set_url, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, t.Name, t.ClientID, t.AuthURL, t.TargetLinkURL, t.TokenURL, t.KeySetURL, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	t.ID = id
	t.CreatedAt = now
	return id, nil
}

func (r *SQLiteRepo) DeleteToolByID(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tools WHERE id = ?`, id)
	return err
}

func (r *SQLiteRepo) ListTools(ctx context.Context) ([]*repoIface.Tool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, client_id, auth_url, target_link_url, token_url, key_set_url, created_at FROM tools ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*repoIface.Tool
	for rows.Next() {
		var t repoIface.Tool
		var created time.Time
		if err := rows.Scan(&t.ID, &t.Name, &t.ClientID, &t.AuthURL, &t.TargetLinkURL, &t.TokenURL, &t.KeySetURL, &created); err != nil {
			return nil, err
		}
		t.CreatedAt = created
		out = append(out, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SQLiteRepo) GetToolByID(ctx context.Context, id int64) (*repoIface.Tool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, client_id, auth_url, target_link_url, token_url, key_set_url, created_at FROM tools WHERE id = ?`, id)
	var t repoIface.Tool
	var created time.Time
	if err := row.Scan(&t.ID, &t.Name, &t.ClientID, &t.AuthURL, &t.TargetLinkURL, &t.TokenURL, &t.KeySetURL, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt = created
	return &t, nil
}

// GetToolByClientID returns a tool by client_id.
func (r *SQLiteRepo) GetToolByClientID(ctx context.Context, clientID string) (*repoIface.Tool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, client_id, auth_url, target_link_url, token_url, key_set_url, created_at FROM tools WHERE client_id = ?`, clientID)
	var t repoIface.Tool
	var created time.Time
	if err := row.Scan(&t.ID, &t.Name, &t.ClientID, &t.AuthURL, &t.TargetLinkURL, &t.TokenURL, &t.KeySetURL, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt = created
	return &t, nil
}

// TryUseClientAssertionJTI records a client_assertion jti if it does not already exist and is not expired.
// Returns true if newly inserted, false if replay (already present and not expired).
func (r *SQLiteRepo) TryUseClientAssertionJTI(ctx context.Context, jti string, clientID string, exp time.Time) (bool, error) {
    // Clean up any expired JTI with same id first (optional)
    // Insert-if-not-exists pattern
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return false, err
    }
    defer func() {
        _ = tx.Rollback()
    }()

    // Check existing
    var existingExp time.Time
    err = tx.QueryRowContext(ctx, `SELECT expires_at FROM client_assertion_jtis WHERE jti = ?`, jti).Scan(&existingExp)
    if err != nil && err != sql.ErrNoRows {
        return false, err
    }
    if err == nil {
        // Exists: treat as replay if now is before existing expiration
        if time.Now().Before(existingExp) {
            return false, nil
        }
        // Expired -> replace
        if _, err := tx.ExecContext(ctx, `DELETE FROM client_assertion_jtis WHERE jti = ?`, jti); err != nil {
            return false, err
        }
    }
    if _, err := tx.ExecContext(ctx, `INSERT INTO client_assertion_jtis (jti, client_id, expires_at) VALUES (?, ?, ?)`, jti, clientID, exp.UTC()); err != nil {
        return false, err
    }
    if err := tx.Commit(); err != nil {
        return false, err
    }
    return true, nil
}
