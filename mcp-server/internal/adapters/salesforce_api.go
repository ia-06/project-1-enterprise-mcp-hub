// Package adapters — Salesforce REST API adapter with OAuth2 ROPC token flow
// and Supabase cache fallback (System Resilience Matrix — Scenario A).
//
// # Authentication
//
// This adapter uses the OAuth2 Resource Owner Password Credentials (ROPC) flow:
//
//  1. POST to <SF_LOGIN_URL>/services/oauth2/token with grant_type=password
//     and the Connected App credentials + username/password.
//  2. Receive a short-lived access_token and instance_url.
//  3. Use the access_token as a Bearer token for all subsequent API calls.
//
// The token is refreshed lazily on every GetAccount call (Salesforce tokens
// are valid for 2 hours by default; for a low-traffic dev hub this is fine).
// A production system would cache and proactively refresh the token.
//
// # Resilience
//
// Resolution order for GetAccount:
//  1. If SF_USE_MOCK=true  → return nil (no fixtures).
//  2. Obtain OAuth2 bearer token.
//  3. Call Salesforce REST API with a context deadline.
//  4. On timeout or 5xx  → attempt Supabase cache lookup (if enabled).
//  5. On cache hit       → return Account with Source="cache".
//  6. On cache miss      → return ErrSalesforceUnavailable.
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/cache"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// ErrSalesforceUnavailable is the sentinel error returned when Salesforce is
// unreachable AND the cache fallback either fails or is disabled.
// The JSON-RPC handler maps this to error code -32002.
var ErrSalesforceUnavailable = errors.New("salesforce unreachable")

// ---------------------------------------------------------------------------
// Adapter
// ---------------------------------------------------------------------------

// SalesforceAdapter wraps the Salesforce REST API with an OAuth2 token cache
// and a Supabase fallback for the Scenario A resilience path.
type SalesforceAdapter struct {
	cfg    config.Config
	client *http.Client
	cache  cache.AccountCache

	// token guards the cached OAuth2 bearer token (lazy refresh per call).
	mu          sync.Mutex
	accessToken string
	instanceURL string
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

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Ping checks Salesforce connectivity.
// In mock mode it always returns nil.
// In live mode it obtains (or reuses) an OAuth token; a successful token
// exchange is treated as a connectivity confirmation.
func (a *SalesforceAdapter) Ping(ctx context.Context) error {
	if a.cfg.SfUseMock {
		return nil
	}
	_, _, err := a.ensureToken(ctx)
	return err
}

// GetAccount fetches a Salesforce Account by its 15/18-character Record ID.
//
// Resolution order:
//  1. If SF_USE_MOCK=true  → return nil account (no fixture data).
//  2. Obtain OAuth2 bearer token.
//  3. Call Salesforce REST API.
//  4. On failure          → attempt Supabase cache lookup.
//  5. On cache hit        → return Account with Source="cache".
//  6. On cache miss       → return ErrSalesforceUnavailable.
func (a *SalesforceAdapter) GetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	if a.cfg.SfUseMock {
		// Mock mode: no fixture data. Return nil so callers degrade gracefully.
		return nil, fmt.Errorf("mock: account lookup disabled in live-migration mode")
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
		// Cache miss or disabled — surface unavailable.
		return nil, ErrSalesforceUnavailable
	}

	cached.Source = "cache"
	return cached, nil
}

// ---------------------------------------------------------------------------
// OAuth2 ROPC token flow
// ---------------------------------------------------------------------------

// sfTokenResponse is the JSON response from the Salesforce OAuth2 token endpoint.
type sfTokenResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// ensureToken lazily fetches a fresh OAuth2 bearer token using the Resource
// Owner Password Credentials flow. The token is NOT cached between calls —
// for a production system you would store it with an expiry check.
// Returns (accessToken, instanceURL, error).
func (a *SalesforceAdapter) ensureToken(ctx context.Context) (string, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// For simplicity in this implementation we always fetch a fresh token.
	// A production upgrade would check token expiry and skip the roundtrip.
	tokenURL := strings.TrimRight(a.cfg.SfLoginURL, "/") + "/services/oauth2/token"

	body := url.Values{}
	body.Set("grant_type", "password")
	body.Set("client_id", a.cfg.SfClientID)
	body.Set("client_secret", a.cfg.SfClientSecret)
	body.Set("username", a.cfg.SfUsername)
	body.Set("password", a.cfg.SfPassword) // password + security token concatenated

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("salesforce: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("salesforce: token request HTTP: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("salesforce: read token response: %w", err)
	}

	var tok sfTokenResponse
	if err := json.Unmarshal(rawBody, &tok); err != nil {
		return "", "", fmt.Errorf("salesforce: unmarshal token response: %w", err)
	}

	if tok.Error != "" {
		return "", "", fmt.Errorf("salesforce OAuth2 error: %s — %s", tok.Error, tok.ErrorDesc)
	}

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("salesforce: token HTTP %d", resp.StatusCode)
	}

	a.accessToken = tok.AccessToken
	a.instanceURL = tok.InstanceURL
	return a.accessToken, a.instanceURL, nil
}

// ---------------------------------------------------------------------------
// Live Salesforce REST implementation
// ---------------------------------------------------------------------------

// sfAccountResponse is a minimal mapping of the Salesforce Account sObject.
type sfAccountResponse struct {
	ID            string      `json:"Id"`
	Name          string      `json:"Name"`
	Type          string      `json:"Type"`
	AnnualRevenue json.Number `json:"AnnualRevenue"`
	Industry      string      `json:"Industry"`
	OwnerID       string      `json:"OwnerId"`
	HealthScore   json.Number `json:"Account_Health_Score__c"`
}

func (a *SalesforceAdapter) liveGetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	accessToken, instanceURL, err := a.ensureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("salesforce: token acquisition failed: %w", err)
	}

	fields := "Id,Name,Type,AnnualRevenue,Industry,OwnerId,Account_Health_Score__c"
	endpoint := fmt.Sprintf(
		"%s/services/data/%s/sobjects/Account/%s?fields=%s",
		strings.TrimRight(instanceURL, "/"),
		a.cfg.SfAPIVersion,
		accountID,
		fields,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("salesforce: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		// Network timeout or connection refused — trigger fallback.
		return nil, fmt.Errorf("salesforce: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("salesforce: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// 4xx / 5xx — trigger fallback.
		return nil, fmt.Errorf("salesforce: API error HTTP %d — %s", resp.StatusCode, string(body))
	}

	var sfResp sfAccountResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		return nil, fmt.Errorf("salesforce: unmarshal response: %w", err)
	}

	healthScore, _ := sfResp.HealthScore.Float64()
	revenue, _ := sfResp.AnnualRevenue.Int64()

	return &domain.Account{
		ID:          sfResp.ID,
		Name:        sfResp.Name,
		Tier:        sfResp.Type,
		MRRCents:    revenue / 12, // rough monthly approximation from annual revenue
		HealthScore: healthScore,
		Owner:       sfResp.OwnerID,
		Industry:    sfResp.Industry,
		Source:      "live",
	}, nil
}
