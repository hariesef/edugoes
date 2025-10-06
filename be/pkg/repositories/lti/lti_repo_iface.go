package lti

import (
	"context"
	"time"
)

// Tool represents an LTI tool registration record.
type Tool struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	ClientID        string    `json:"client_id"`
	AuthURL         string    `json:"auth_url"`
	TargetLinkURL   string    `json:"target_link_url"`
	TargetLaunchURL string    `json:"target_launch_url"`
	KeySetURL       string    `json:"key_set_url"`
	CreatedAt       time.Time `json:"created_at"`
}

// DeepLinkSelection represents a persisted deep-link content item for reuse.
type DeepLinkSelection struct {
	ID              int64     `json:"id"`
	ClientID        string    `json:"client_id"`
	ToolName        string    `json:"tool_name"`
	URL             string    `json:"url"`
	ContentItemJSON string    `json:"content_item_json"`
	CreatedAt       time.Time `json:"created_at"`
}

// Repository defines storage operations for LTI data.
type Repository interface {
	// Health is a simple check to verify repository works.
	Health() error
	// Disconnect gracefully closes resources. Should be safe to call on shutdown.
	Disconnect()
	// RegisterTool inserts a new tool registration and returns its ID.
	RegisterTool(ctx context.Context, t *Tool) (int64, error)
	// ListTools returns all registered tools.
	ListTools(ctx context.Context) ([]*Tool, error)
	// GetToolByClientID returns a tool by its client_id.
	GetToolByClientID(ctx context.Context, clientID string) (*Tool, error)
	// GetToolByID returns a tool by its ID.
	GetToolByID(ctx context.Context, id int64) (*Tool, error)
	// DeleteToolByID deletes a tool by its numeric ID.
	DeleteToolByID(ctx context.Context, id int64) error

	// Persisted deep link selections
	CreateDeepLinkSelection(ctx context.Context, sel *DeepLinkSelection) (int64, error)
	ListDeepLinkSelections(ctx context.Context) ([]*DeepLinkSelection, error)
	GetDeepLinkSelection(ctx context.Context, id int64) (*DeepLinkSelection, error)
	DeleteDeepLinkSelection(ctx context.Context, id int64) error
}
