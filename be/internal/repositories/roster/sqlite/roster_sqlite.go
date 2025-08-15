package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"

	r "github.com/quipper/poc/lti/be/pkg/repositories/roster"
)

type SQLiteRepo struct{ db *sql.DB }

func NewSQLiteRepo(path string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil { return nil, err }
	if err := initSchema(db); err != nil { _ = db.Close(); return nil, err }
	return &SQLiteRepo{db: db}, nil
}

func (s *SQLiteRepo) Disconnect() { _ = s.db.Close() }

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS members (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  context_id TEXT NOT NULL,
	  user_id TEXT NOT NULL,
	  name TEXT,
	  given_name TEXT,
	  family_name TEXT,
	  email TEXT,
	  roles_json TEXT,
	  status TEXT,
	  updated_at TIMESTAMP NOT NULL,
	  UNIQUE(context_id, user_id)
	);
	`)
	return err
}

func (s *SQLiteRepo) ListMembers(ctx context.Context, contextID string) ([]*r.Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, name, given_name, family_name, email, roles_json, status, updated_at FROM members WHERE context_id = ? ORDER BY user_id ASC`, contextID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []*r.Member
	for rows.Next() {
		var m r.Member
		var rolesStr sql.NullString
		var name, given, family, email sql.NullString
		var status sql.NullString
		var ts time.Time
		if err := rows.Scan(&m.UserID, &name, &given, &family, &email, &rolesStr, &status, &ts); err != nil { return nil, err }
		if name.Valid { m.Name = name.String }
		if given.Valid { m.GivenName = given.String }
		if family.Valid { m.FamilyName = family.String }
		if email.Valid { m.Email = email.String }
		if status.Valid { m.Status = status.String }
		if rolesStr.Valid && rolesStr.String != "" {
			_ = json.Unmarshal([]byte(rolesStr.String), &m.Roles)
		}
		m.UpdatedAt = ts
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (s *SQLiteRepo) ListMembersPage(ctx context.Context, contextID string, offset, limit int) ([]*r.Member, int, error) {
    // total count
    var total int
    if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE context_id = ?`, contextID).Scan(&total); err != nil {
        return nil, 0, err
    }
    // page query
    rows, err := s.db.QueryContext(ctx, `SELECT user_id, name, given_name, family_name, email, roles_json, status, updated_at FROM members WHERE context_id = ? ORDER BY id ASC LIMIT ? OFFSET ?`, contextID, limit, offset)
    if err != nil { return nil, 0, err }
    defer rows.Close()
    var out []*r.Member
    for rows.Next() {
        var m r.Member
        var rolesStr sql.NullString
        var name, given, family, email sql.NullString
        var status sql.NullString
        var ts time.Time
        if err := rows.Scan(&m.UserID, &name, &given, &family, &email, &rolesStr, &status, &ts); err != nil { return nil, 0, err }
        if name.Valid { m.Name = name.String }
        if given.Valid { m.GivenName = given.String }
        if family.Valid { m.FamilyName = family.String }
        if email.Valid { m.Email = email.String }
        if status.Valid { m.Status = status.String }
        if rolesStr.Valid && rolesStr.String != "" {
            _ = json.Unmarshal([]byte(rolesStr.String), &m.Roles)
        }
        m.UpdatedAt = ts
        out = append(out, &m)
    }
    if err := rows.Err(); err != nil { return nil, 0, err }
    return out, total, nil
}

func (s *SQLiteRepo) UpsertMember(ctx context.Context, contextID string, m *r.Member) error {
	rolesJSON := "[]"
	if b, err := json.Marshal(m.Roles); err == nil { rolesJSON = string(b) }
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
	INSERT INTO members (context_id, user_id, name, given_name, family_name, email, roles_json, status, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(context_id, user_id)
	DO UPDATE SET name = excluded.name, given_name = excluded.given_name, family_name = excluded.family_name, email = excluded.email, roles_json = excluded.roles_json, status = excluded.status, updated_at = excluded.updated_at
	`, contextID, m.UserID, m.Name, m.GivenName, m.FamilyName, m.Email, rolesJSON, m.Status, now)
	if err == nil { m.UpdatedAt = now }
	return err
}

func (s *SQLiteRepo) DeleteMember(ctx context.Context, contextID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM members WHERE context_id = ? AND user_id = ?`, contextID, userID)
	return err
}
