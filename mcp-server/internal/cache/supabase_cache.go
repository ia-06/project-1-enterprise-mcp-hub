// Package cache provides the Supabase PostgREST-backed fallback cache used
// when Salesforce is unreachable (System Resilience Matrix — Scenario A).
//
// The cache reads from the `cache_accounts` table, which mirrors the
// Salesforce Account sObject schema. Only read operations are performed here.
//
// Authentication uses the Supabase anon key for read-only PostgREST access.
// The service_role key (SUPABASE_SERVICE_ROLE_KEY) is loaded into Config for
// future admin/write operations but is NOT used in this read-only path.
//
// This file performs real outbound HTTPS requests to the live Supabase
// cluster — there is no mock path. If SUPABASE_ENABLED=false or credentials
// are missing, every call returns ErrCacheDisabled immediately.
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

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrCacheDisabled is returned when Supabase caching is not enabled or
// the required credentials are absent.
var ErrCacheDisabled = errors.New("supabase cache disabled")

// ErrCacheMiss is returned when the requested account is not in the cache table.
var ErrCacheMiss = errors.New("account not found in supabase cache")

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// AccountCache defines read-only operations on the Supabase account cache.
type AccountCache interface {
	GetAccount(ctx context.Context, sfID string) (*domain.Account, error)
}

// ---------------------------------------------------------------------------
// Supabase PostgREST implementation
// ---------------------------------------------------------------------------

// supabaseAccountCache implements AccountCache using Supabase's auto-generated
// PostgREST REST API, accessed over HTTPS with the project anon key.
type supabaseAccountCache struct {
	cfg    config.Config
	client *http.Client
}

// NewSupabaseAccountCache creates a cache client backed by the live Supabase
// cluster. If SUPABASE_ENABLED=false, all calls immediately return ErrCacheDisabled.
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
// Returns ErrCacheDisabled immediately when:
//   - SUPABASE_ENABLED=false
//   - SUPABASE_URL or SUPABASE_ANON_KEY are empty
//
// Returns ErrCacheMiss when no row matches sfID.
func (c *supabaseAccountCache) GetAccount(ctx context.Context, sfID string) (*domain.Account, error) {
	if !c.cfg.SupabaseEnabled {
		return nil, ErrCacheDisabled
	}
	if c.cfg.SupabaseURL == "" || c.cfg.SupabaseAnonKey == "" {
		return nil, ErrCacheDisabled
	}
	if sfID == "" {
		return nil, fmt.Errorf("supabase cache: sfID must not be empty")
	}

	// PostgREST filter: exact match on the sf_id column.
	endpoint := fmt.Sprintf(
		"%s/rest/v1/cache_accounts?sf_id=eq.%s&select=sf_id,name,tier,mrr_cents,health_score,owner,industry",
		c.cfg.SupabaseURL,
		sfID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("supabase: build request: %w", err)
	}

	// PostgREST requires both the apikey header and a Bearer Authorization
	// header. The anon key is used for read-only table access.
	req.Header.Set("apikey", c.cfg.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+c.cfg.SupabaseAnonKey)
	req.Header.Set("Accept", "application/json")
	// Request a single object instead of an array when we expect exactly one row.
	// "Accept: application/vnd.pgrst.object+json" would 406 if 0 rows — so we
	// keep the default array response and check length ourselves.

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("supabase: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("supabase: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("supabase: API error HTTP %d — %s", resp.StatusCode, string(body))
	}

	// PostgREST returns a JSON array even for single-row filters.
	var rows []supabaseCacheRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("supabase: unmarshal response: %w", err)
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
		// Source is intentionally blank here; the caller (SalesforceAdapter)
		// sets it to "cache" so the UI renders the correct resilience badge.
	}, nil
}

// ---------------------------------------------------------------------------
// Supabase response row shape
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
