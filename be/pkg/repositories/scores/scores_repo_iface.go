package scores

import (
	"context"
	"time"
)

// LineItem represents an AGS line item within a specific context (e.g., course).
type LineItem struct {
	ID             int64      `json:"id"`
	ContextID      string     `json:"-"` // context conveyed via URL path; not part of payload
	Label          string     `json:"label"`
	ResourceID     string     `json:"resourceId,omitempty"`
	ResourceLinkID string     `json:"resourceLinkId,omitempty"`
	Tag            string     `json:"tag,omitempty"`
	ScoreMaximum   float64    `json:"scoreMaximum"`
	StartAt        *time.Time `json:"startDateTime,omitempty"`
	EndAt          *time.Time `json:"endDateTime,omitempty"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`
}

// Score is the POST payload to record a user's score. This is used to upsert a Result.
type Score struct {
	UserID       string    `json:"userId"`
	Timestamp    time.Time `json:"timestamp"`
	ScoreGiven   *float64  `json:"scoreGiven,omitempty"`
	ScoreMaximum *float64  `json:"scoreMaximum,omitempty"`
	Comment      string    `json:"comment,omitempty"`
	// LTI AGS required status fields
	ActivityProgress string `json:"activityProgress,omitempty"`
	GradingProgress  string `json:"gradingProgress,omitempty"`
}

// Result represents the latest computed result per user for a line item.
type Result struct {
	UserID           string    `json:"userId"`
	ResultScore      *float64  `json:"resultScore,omitempty"`
	ResultMaximum    *float64  `json:"resultMaximum,omitempty"`
	Comment          string    `json:"comment,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
	ActivityProgress string    `json:"activityProgress,omitempty"`
	GradingProgress  string    `json:"gradingProgress,omitempty"`
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

	// CreateLineItemMapping creates a one-to-one mapping between a lineItemId and a resourceLinkId.
	// Both lineItemId and resourceLinkId must be globally unique across the mapping table.
	CreateLineItemMapping(ctx context.Context, lineItemID int64, resourceLinkID string) error

	// GetLineItemIDByResourceLinkID returns the lineItemId mapped to the given resourceLinkId.
	// If no mapping exists, returns 0 and a nil error.
	GetLineItemIDByResourceLinkID(ctx context.Context, resourceLinkID string) (int64, error)
}
