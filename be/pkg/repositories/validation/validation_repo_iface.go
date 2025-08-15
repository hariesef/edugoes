package validation

import (
    "context"
    "time"
)

// Repository defines storage needed for security validation concerns such as
// client_assertion JTI replay protection and OIDC state/nonce persistence.
type Repository interface {
    // TryUseClientAssertionJTI attempts to record a client_assertion jti for replay protection.
    // It should return true if the jti was newly recorded, false if it already existed (replay),
    // and an error for storage issues.
    TryUseClientAssertionJTI(ctx context.Context, jti string, clientID string, exp time.Time) (bool, error)

    // CreateOIDCState stores an OIDC state with optional metadata and expiry.
    // contextID is used later in OIDC to populate AGS claims; resourceLinkID identifies the content item.
    CreateOIDCState(ctx context.Context, state string, clientID string, targetLinkURI string, contextID string, resourceLinkID string, exp time.Time) error
    // ConsumeOIDCState atomically loads and invalidates a state, returning its data.
    // ok=false if not found or already used/expired.
    ConsumeOIDCState(ctx context.Context, state string) (clientID string, targetLinkURI string, resourceLinkID string, contextID string, ok bool, err error)
}
