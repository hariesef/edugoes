package roster

import (
	"context"
	"time"
)

// Member represents an NRPS membership in a context.
// Roles follow LTI role URIs; store as array of strings.
// Status can be Active or Inactive.
type Member struct {
	UserID     string    `json:"user_id"`
	Name       string    `json:"name,omitempty"`
	GivenName  string    `json:"given_name,omitempty"`
	FamilyName string    `json:"family_name,omitempty"`
	Email      string    `json:"email,omitempty"`
	Roles      []string  `json:"roles,omitempty"`
	Status     string    `json:"status,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Repository interface {
    // ListMembersPage returns members for a context with pagination,
    // along with the total count for the context.
    ListMembersPage(ctx context.Context, contextID string, offset, limit int) ([]*Member, int, error)
    UpsertMember(ctx context.Context, contextID string, m *Member) error
    DeleteMember(ctx context.Context, contextID, userID string) error
    Disconnect()
}
