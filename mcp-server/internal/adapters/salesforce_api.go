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

// ListAccounts returns up to `limit` Salesforce Account records ordered by
// Name. If limit is 0 or negative, it defaults to 50. Maximum is 200.
// Used by the Next.js dashboard to populate the dynamic account selector.
func (a *SalesforceAdapter) ListAccounts(ctx context.Context, limit int) ([]domain.Account, error) {
	if a.cfg.SfUseMock {
		return []domain.Account{}, nil
	}
	return a.liveListAccounts(ctx, limit)
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
// HealthScore uses a default value because the custom field Account_Health_Score__c
// is not created in standard/developer orgs; requesting it would cause a 400 INVALID_FIELD.
type sfAccountResponse struct {
	ID            string      `json:"Id"`
	Name          string      `json:"Name"`
	Type          string      `json:"Type"`
	AnnualRevenue json.Number `json:"AnnualRevenue"`
	Industry      string      `json:"Industry"`
	OwnerID       string      `json:"OwnerId"`
}

func (a *SalesforceAdapter) liveGetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	accessToken, instanceURL, err := a.ensureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("salesforce: token acquisition failed: %w", err)
	}

	fields := "Id,Name,Type,AnnualRevenue,Industry,OwnerId"
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
		return nil, fmt.Errorf("salesforce: API error HTTP %d — %s", resp.StatusCode, string(body))
	}

	var sfResp sfAccountResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		return nil, fmt.Errorf("salesforce: unmarshal response: %w", err)
	}

	// HealthScore defaults to 85 — Account_Health_Score__c is not a standard
	// Salesforce field and is not requested to avoid INVALID_FIELD errors.
	const defaultHealthScore = 85.0
	revenue, _ := sfResp.AnnualRevenue.Int64()

	return &domain.Account{
		ID:          sfResp.ID,
		Name:        sfResp.Name,
		Tier:        sfResp.Type,
		MRRCents:    revenue / 12, // rough monthly approximation from annual revenue
		HealthScore: defaultHealthScore,
		Owner:       sfResp.OwnerID,
		Industry:    sfResp.Industry,
		Source:      "live",
	}, nil
}

// ---------------------------------------------------------------------------
// ListAccounts — SOQL query implementation
// ---------------------------------------------------------------------------

// sfQueryResponse is the JSON envelope returned by the Salesforce Query API.
type sfQueryResponse struct {
	TotalSize int                  `json:"totalSize"`
	Done      bool                 `json:"done"`
	Records   []sfAccountListRecord `json:"records"`
}

// sfAccountListRecord is a lightweight Account record for the list view.
type sfAccountListRecord struct {
	ID            string      `json:"Id"`
	Name          string      `json:"Name"`
	Type          string      `json:"Type"`
	Industry      string      `json:"Industry"`
	OwnerID       string      `json:"OwnerId"`
	AnnualRevenue json.Number `json:"AnnualRevenue"`
}

func (a *SalesforceAdapter) liveListAccounts(ctx context.Context, limit int) ([]domain.Account, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	accessToken, instanceURL, err := a.ensureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("salesforce: token acquisition failed: %w", err)
	}

	// SOQL query — include AnnualRevenue so MRR can be approximated for list view.
	soql := fmt.Sprintf(
		"SELECT Id,Name,Type,Industry,OwnerId,AnnualRevenue FROM Account ORDER BY Name ASC LIMIT %d",
		limit,
	)
	endpoint := fmt.Sprintf(
		"%s/services/data/%s/query?q=%s",
		strings.TrimRight(instanceURL, "/"),
		a.cfg.SfAPIVersion,
		url.QueryEscape(soql),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("salesforce listAccounts: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("salesforce listAccounts: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("salesforce listAccounts: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("salesforce listAccounts: API error HTTP %d — %s", resp.StatusCode, string(body))
	}

	var queryResp sfQueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("salesforce listAccounts: unmarshal response: %w", err)
	}

	accounts := make([]domain.Account, 0, len(queryResp.Records))
	for _, rec := range queryResp.Records {
		annualRevenue, _ := rec.AnnualRevenue.Int64()
		accounts = append(accounts, domain.Account{
			ID:          rec.ID,
			Name:        rec.Name,
			Tier:        rec.Type,
			Industry:    rec.Industry,
			Owner:       rec.OwnerID,
			MRRCents:    annualRevenue / 12, // approximate monthly from annual
			HealthScore: 85.0,               // default score; full detail available via getAccount
			Source:      "live",
		})
	}
	return accounts, nil
}

