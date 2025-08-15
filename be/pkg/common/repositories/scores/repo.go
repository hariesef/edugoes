package scores

import (
	"context"
	"time"
)

// LineItem represents an AGS line item within a specific context (e.g., course).
type LineItem struct {
	ID             int64      `json:"id"`
	ContextID      string     `json:"context_id"`
	Label          string     `json:"label"`
	ResourceID     string     `json:"resource_id,omitempty"`
	ResourceLinkID string     `json:"resource_link_id,omitempty"`
	Tag            string     `json:"tag,omitempty"`
	ScoreMaximum   float64    `json:"score_maximum"`
	StartAt        *time.Time `json:"start_at,omitempty"`
	EndAt          *time.Time `json:"end_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Score is the POST payload to record a user's score. This is used to upsert a Result.
type Score struct {
	UserID       string     `json:"userId"`
	Timestamp    time.Time  `json:"timestamp"`
	ScoreGiven   *float64   `json:"scoreGiven,omitempty"`
	ScoreMaximum *float64   `json:"scoreMaximum,omitempty"`
	Comment      string     `json:"comment,omitempty"`
}

// Result represents the latest computed result per user for a line item.
type Result struct {
	UserID        string    `json:"userId"`
	ResultScore   *float64  `json:"resultScore,omitempty"`
	ResultMaximum *float64  `json:"resultMaximum,omitempty"`
	Comment       string    `json:"comment,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Repository defines persistence for AGS entities.
type Repository interface {
	CreateLineItem(ctx context.Context, li *LineItem) (int64, error)
	ListLineItems(ctx context.Context, contextID string) ([]*LineItem, error)
	GetLineItem(ctx context.Context, id int64, contextID string) (*LineItem, error)
	UpdateLineItem(ctx context.Context, li *LineItem) error
	DeleteLineItem(ctx context.Context, id int64, contextID string) error

	UpsertResultFromScore(ctx context.Context, lineItemID int64, contextID string, s *Score) error
	ListResultsByLineItem(ctx context.Context, lineItemID int64, contextID string) ([]*Result, error)
}
