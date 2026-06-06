// Package adapters — Salesforce REST adapter with Supabase fallback engine.
//
// System Resilience Matrix — Scenario A implementation:
//   - Primary path: Calls Salesforce REST API with a context deadline derived
//     from GO_REQUEST_TIMEOUT_MS.
//   - On timeout or 5xx: Sets ErrSalesforceUnavailable.
//   - Fallback path: If SUPABASE_ENABLED=true, queries the Supabase cache via
//     the AccountCache interface. On cache hit, returns Account with Source="cache".
//   - On cache miss: Returns ErrSalesforceUnavailable to caller.
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/cache"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// ErrSalesforceUnavailable is the sentinel error returned when Salesforce is
// unreachable AND the cache fallback either fails or is disabled.
// The JSON-RPC handler maps this to error code -32002.
var ErrSalesforceUnavailable = errors.New("salesforce unreachable")

// SalesforceAdapter wraps the Salesforce REST API (or its mock) with a
// Supabase cache fallback for the Scenario A resilience path.
type SalesforceAdapter struct {
	cfg    config.Config
	client *http.Client
	cache  cache.AccountCache
}

// NewSalesforceAdapter creates a new SalesforceAdapter.
func NewSalesforceAdapter(cfg config.Config, accountCache cache.AccountCache) *SalesforceAdapter {
	return &SalesforceAdapter{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		cache: accountCache,
	}
}

// Ping checks Salesforce connectivity. In mock mode it always returns nil.
func (a *SalesforceAdapter) Ping(_ context.Context) error {
	if a.cfg.SfUseMock {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, a.cfg.SfBaseURL+"/services/data/v59.0/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.SfAPIToken)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("salesforce ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

// GetAccount fetches a Salesforce Account by ID.
//
// Resolution order:
//  1. If SF_USE_MOCK=true → return mock fixture data.
//  2. Call Salesforce REST API with context deadline.
//  3. On error (timeout / 5xx) → attempt Supabase cache lookup.
//  4. On cache hit → return Account with Source="cache".
//  5. On cache miss → return ErrSalesforceUnavailable.
func (a *SalesforceAdapter) GetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	if a.cfg.SfUseMock {
		acc := a.mockAccount(accountID)
		if acc == nil {
			return nil, fmt.Errorf("mock: account %q not found", accountID)
		}
		return acc, nil
	}

	// -- Live Salesforce call --
	acc, err := a.liveGetAccount(ctx, accountID)
	if err == nil {
		return acc, nil
	}

	// -- Fallback: Supabase cache --
	if !a.cfg.SupabaseEnabled {
		return nil, ErrSalesforceUnavailable
	}

	cached, cacheErr := a.cache.GetAccount(ctx, accountID)
	if cacheErr != nil {
		// Cache miss or disabled — surface the original Salesforce error.
		return nil, ErrSalesforceUnavailable
	}

	cached.Source = "cache"
	return cached, nil
}

// ---------------------------------------------------------------------------
// Mock implementation (static fixture data)
// ---------------------------------------------------------------------------

func (a *SalesforceAdapter) mockAccount(accountID string) *domain.Account {
	fixtures := map[string]*domain.Account{
		"001ACME000000001": {ID: "001ACME000000001", Name: "ACME Corporation", Tier: "Enterprise", MRRCents: 625000, HealthScore: 87.5, Owner: "Jane Smith", Industry: "Manufacturing", Source: "live"},
		"001BETA000000002": {ID: "001BETA000000002", Name: "Beta Technologies Inc.", Tier: "Mid-Market", MRRCents: 145000, HealthScore: 72.0, Owner: "Bob Johnson", Industry: "Software", Source: "live"},
		"001GAMA000000003": {ID: "001GAMA000000003", Name: "Gamma Retail Group", Tier: "SMB", MRRCents: 42000, HealthScore: 55.3, Owner: "Carol Williams", Industry: "Retail", Source: "live"},
		"001DELT000000004": {ID: "001DELT000000004", Name: "Delta Financial Services", Tier: "Enterprise", MRRCents: 1200000, HealthScore: 94.1, Owner: "Alice Chen", Industry: "Financial Services", Source: "live"},
		"001EPSI000000005": {ID: "001EPSI000000005", Name: "Epsilon Healthcare", Tier: "Mid-Market", MRRCents: 380000, HealthScore: 68.7, Owner: "David Lee", Industry: "Healthcare", Source: "live"},
	}
	return fixtures[accountID]
}

// ---------------------------------------------------------------------------
// Live Salesforce REST implementation
// ---------------------------------------------------------------------------

// sfAccountResponse is a minimal mapping of the Salesforce Account sobject.
type sfAccountResponse struct {
	ID             string      `json:"Id"`
	Name           string      `json:"Name"`
	Type           string      `json:"Type"`
	AnnualRevenue  json.Number `json:"AnnualRevenue"`
	Industry       string      `json:"Industry"`
	OwnerId        string      `json:"OwnerId"`
	AccountHealth  json.Number `json:"Account_Health_Score__c"`
}

func (a *SalesforceAdapter) liveGetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	fields := "Id,Name,Type,AnnualRevenue,Industry,OwnerId,Account_Health_Score__c"
	endpoint := fmt.Sprintf(
		"%s/services/data/v59.0/sobjects/Account/%s?fields=%s",
		strings.TrimRight(a.cfg.SfBaseURL, "/"),
		accountID,
		fields,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("salesforce request build: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.SfAPIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Network timeout or connection refused — trigger fallback.
		return nil, fmt.Errorf("salesforce HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("salesforce response read: %w", err)
	}

	if resp.StatusCode >= 400 {
		// 4xx / 5xx — trigger fallback.
		return nil, fmt.Errorf("salesforce API error: HTTP %d", resp.StatusCode)
	}

	var sfResp sfAccountResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		return nil, fmt.Errorf("salesforce response unmarshal: %w", err)
	}

	healthScore, _ := sfResp.AccountHealth.Float64()
	revenue, _ := sfResp.AnnualRevenue.Int64()

	return &domain.Account{
		ID:          sfResp.ID,
		Name:        sfResp.Name,
		Tier:        sfResp.Type,
		MRRCents:    revenue / 12, // rough monthly approximation
		HealthScore: healthScore,
		Owner:       sfResp.OwnerId,
		Industry:    sfResp.Industry,
		Source:      "live",
	}, nil
}
