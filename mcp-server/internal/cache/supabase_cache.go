// Package cache provides the Supabase-based fallback cache used when
// Salesforce is unreachable (System Resilience Matrix — Scenario A).
//
// The cache reads from the `cache_accounts` table which mirrors the
// Salesforce Account sobject. In Phase 1, only read operations are needed.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// ErrCacheDisabled is returned when Supabase caching is not enabled.
var ErrCacheDisabled = errors.New("supabase cache disabled")

// ErrCacheMiss is returned when the requested account is not in the cache.
var ErrCacheMiss = errors.New("account not found in cache")

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// AccountCache defines the read-only operations on the Supabase account cache.
type AccountCache interface {
	GetAccount(ctx context.Context, sfID string) (*domain.Account, error)
}

// ---------------------------------------------------------------------------
// Supabase HTTP implementation
// ---------------------------------------------------------------------------

// supabaseAccountCache implements AccountCache using Supabase's REST API
// (PostgREST), accessed via the anon key over HTTPS.
type supabaseAccountCache struct {
	cfg    config.Config
	client *http.Client
}

// NewSupabaseAccountCache creates a cache client. If SUPABASE_ENABLED=false,
// all calls immediately return ErrCacheDisabled.
func NewSupabaseAccountCache(cfg config.Config) AccountCache {
	return &supabaseAccountCache{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// GetAccount retrieves a cached Salesforce Account snapshot from Supabase.
//
// Uses the Supabase PostgREST endpoint:
//
//	GET <SUPABASE_URL>/rest/v1/cache_accounts?sf_id=eq.<sfID>&select=*
//
// Returns ErrCacheDisabled if SUPABASE_ENABLED=false.
// Returns ErrCacheMiss if no row matches sfID.
func (c *supabaseAccountCache) GetAccount(ctx context.Context, sfID string) (*domain.Account, error) {
	if !c.cfg.SupabaseEnabled {
		return nil, ErrCacheDisabled
	}
	if c.cfg.SupabaseURL == "" || c.cfg.SupabaseAnonKey == "" {
		return nil, ErrCacheDisabled
	}

	endpoint := fmt.Sprintf(
		"%s/rest/v1/cache_accounts?sf_id=eq.%s&select=sf_id,name,tier,mrr_cents,health_score,owner,industry",
		c.cfg.SupabaseURL,
		sfID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("supabase request build: %w", err)
	}
	req.Header.Set("apikey", c.cfg.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+c.cfg.SupabaseAnonKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("supabase HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("supabase response read: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("supabase API error: HTTP %d — %s", resp.StatusCode, string(body))
	}

	// PostgREST returns an array even for a single-row filter.
	var rows []supabaseCacheRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("supabase response unmarshal: %w", err)
	}

	if len(rows) == 0 {
		return nil, ErrCacheMiss
	}

	row := rows[0]
	return &domain.Account{
		ID:          row.SFID,
		Name:        row.Name,
		Tier:        row.Tier,
		MRRCents:    row.MRRCents,
		HealthScore: row.HealthScore,
		Owner:       row.Owner,
		Industry:    row.Industry,
		// Source is intentionally left blank here; the caller sets it to "cache".
	}, nil
}

// ---------------------------------------------------------------------------
// Supabase response row type
// ---------------------------------------------------------------------------

type supabaseCacheRow struct {
	SFID        string  `json:"sf_id"`
	Name        string  `json:"name"`
	Tier        string  `json:"tier"`
	MRRCents    int64   `json:"mrr_cents"`
	HealthScore float64 `json:"health_score"`
	Owner       string  `json:"owner"`
	Industry    string  `json:"industry"`
}
