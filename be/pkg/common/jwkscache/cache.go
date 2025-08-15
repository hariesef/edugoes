package jwkscache

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// Cache provides JWKS retrieval with HTTP caching semantics.
type Cache interface {
	Get(ctx context.Context, url string) (jwk.Set, error)
	Invalidate(url string)
}

// entry stores a cached JWKS and metadata derived from HTTP caching headers.
type entry struct {
	set            jwk.Set
	expiry         time.Time
	allowStaleUntil time.Time
	etag           string
	lastModified   time.Time
}

type memoryCache struct {
	mu         sync.RWMutex
	entries    map[string]*entry
	client     *http.Client
	defaultTTL time.Duration
	staleGrace time.Duration
}

var (
	defaultOnce sync.Once
	defaultC    Cache
)

// Default returns a process-wide JWKS cache with sensible defaults.
func Default() Cache {
	defaultOnce.Do(func() {
		defaultC = New(10*time.Minute, 1*time.Hour)
	})
	return defaultC
}

// New creates a new in-memory JWKS cache.
// defaultTTL is used when the response does not specify caching directives.
// staleGrace allows serving stale content on transient fetch failures.
func New(defaultTTL, staleGrace time.Duration) Cache {
	return &memoryCache{
		entries:    make(map[string]*entry),
		client:     &http.Client{Timeout: 5 * time.Second},
		defaultTTL: defaultTTL,
		staleGrace: staleGrace,
	}
}

func (c *memoryCache) Invalidate(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, url)
}

func (c *memoryCache) Get(ctx context.Context, url string) (jwk.Set, error) {
	// Fast path: return if fresh
	if set := c.getFresh(url); set != nil {
		return set, nil
	}
	// Otherwise, fetch or revalidate
	return c.fetch(ctx, url)
}

func (c *memoryCache) getFresh(url string) jwk.Set {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.entries[url]; ok {
		if time.Now().Before(e.expiry) && e.set != nil {
			return e.set
		}
	}
	return nil
}

func (c *memoryCache) fetch(ctx context.Context, url string) (jwk.Set, error) {
	c.mu.Lock()
	e := c.entries[url]
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Conditional headers
	if e != nil {
		if e.etag != "" {
			req.Header.Set("If-None-Match", e.etag)
		}
		if !e.lastModified.IsZero() {
			req.Header.Set("If-Modified-Since", e.lastModified.UTC().Format(http.TimeFormat))
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// Serve stale if available within grace window
		if e != nil && time.Now().Before(e.allowStaleUntil) && e.set != nil {
			return e.set, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified: // 304
		if e == nil || e.set == nil {
			return nil, errors.New("jwkscache: 304 but no cached entry")
		}
		// Update expiry based on headers
		newExpiry, allowStale := computeExpiry(resp.Header, c.defaultTTL, c.staleGrace)
		c.mu.Lock()
		e.expiry = newExpiry
		e.allowStaleUntil = allowStale
		c.mu.Unlock()
		return e.set, nil
	case http.StatusOK: // 200
		// Read with size guard
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB
		if err != nil {
			return nil, err
		}
		set, err := jwk.Parse(body)
		if err != nil {
			return nil, err
		}
		newE := &entry{set: set}
		newE.expiry, newE.allowStaleUntil = computeExpiry(resp.Header, c.defaultTTL, c.staleGrace)
		newE.etag = resp.Header.Get("ETag")
		if lm := resp.Header.Get("Last-Modified"); lm != "" {
			if t, err := time.Parse(http.TimeFormat, lm); err == nil {
				newE.lastModified = t
			}
		}
		c.mu.Lock()
		c.entries[url] = newE
		c.mu.Unlock()
		return set, nil
	default:
		// Serve stale if possible
		if e != nil && time.Now().Before(e.allowStaleUntil) && e.set != nil {
			return e.set, nil
		}
		return nil, errors.New("jwkscache: unexpected status " + strconv.Itoa(resp.StatusCode))
	}
}

func computeExpiry(h http.Header, defTTL, staleGrace time.Duration) (expiry, allowStaleUntil time.Time) {
	now := time.Now()
	cc := parseCacheControl(h.Get("Cache-Control"))
	if cc["no-store"] == "true" {
		return now, now // immediately expired, no stale allowed
	}
	if maxAge, ok := cc["max-age"]; ok {
		if secs, err := strconv.Atoi(maxAge); err == nil {
			d := time.Duration(secs) * time.Second
			exp := now.Add(d)
			return exp, exp.Add(staleGrace)
		}
	}
	if expStr := h.Get("Expires"); expStr != "" {
		if t, err := time.Parse(http.TimeFormat, expStr); err == nil {
			return t, t.Add(staleGrace)
		}
	}
	// Fallback
	exp := now.Add(defTTL)
	return exp, exp.Add(staleGrace)
}

func parseCacheControl(v string) map[string]string {
	m := map[string]string{}
	for _, part := range strings.Split(v, ",") {
		p := strings.TrimSpace(strings.ToLower(part))
		if p == "" {
			continue
		}
		// we only need flags and max-age
		if strings.HasPrefix(p, "max-age=") {
			m["max-age"] = strings.TrimPrefix(p, "max-age=")
			continue
		}
		m[p] = "true"
	}
	return m
}
