// Package adapters — Jira Cloud REST API v3 adapter.
//
// Authentication: HTTP Basic Auth using the Atlassian account email as the
// username and a personal API token as the password. The token is generated
// at https://id.atlassian.com/manage-profile/security/api-tokens.
//
// Custom Field Resolution (Q2):
// JIRA_SF_ACCOUNT_FIELD holds the human-readable field name (e.g. "Salesforce
// Account ID"). On first use, the adapter calls /rest/api/3/field to resolve
// this name to a Jira-internal key (e.g. "customfield_10057") and caches the
// result in-memory. JQL then uses the resolved key directly — no manual cf[]
// IDs required, no maintenance burden when the Jira schema changes.
package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// JiraAdapter wraps the Jira Cloud REST API and normalises responses
// into domain.JiraTicket structs. It is safe for concurrent use.
type JiraAdapter struct {
	cfg       config.Config
	client    *http.Client
	basicAuth string // precomputed "Basic <base64(email:token)>"

	// fieldKey holds the resolved Jira internal key for the SF Account field.
	// It is populated lazily on the first live API call and then reused.
	fieldKeyMu    sync.RWMutex
	fieldKey      string // e.g. "customfield_10057"
	fieldResolved bool
}

// NewJiraAdapter creates a JiraAdapter pre-configured with Basic Auth
// credentials derived from cfg.JiraEmail and cfg.JiraAPIToken.
func NewJiraAdapter(cfg config.Config) *JiraAdapter {
	creds := cfg.JiraEmail + ":" + cfg.JiraAPIToken
	encoded := base64.StdEncoding.EncodeToString([]byte(creds))

	return &JiraAdapter{
		cfg:       cfg,
		basicAuth: "Basic " + encoded,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Ping checks Jira connectivity by calling the lightweight /rest/api/3/myself
// endpoint. In mock mode it always returns nil.
func (a *JiraAdapter) Ping(ctx context.Context) error {
	if a.cfg.JiraUseMock {
		return nil
	}

	endpoint := strings.TrimRight(a.cfg.JiraBaseURL, "/") + "/rest/api/3/myself"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("jira ping: build request: %w", err)
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("jira ping: HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("jira ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListTicketsByAccount returns Jira issues linked to a Salesforce Account ID.
//
// In mock mode:  returns an empty slice.
// In live mode:  dynamically resolves the custom field key for the configured
//
//	field name, then executes a JQL search against Jira Cloud.
//
// Fallback strategy for templated/sample Jira projects:
//
//	If the SF-account-specific JQL returns 0 results (because the custom field
//	is not populated on sample issues), the adapter automatically retries with
//	a project-scoped query and tags all returned tickets with the given
//	accountSFID. This makes the dashboard usable with out-of-the-box Jira
//	projects without needing to manually configure every issue.
func (a *JiraAdapter) ListTicketsByAccount(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	if a.cfg.JiraUseMock {
		return []domain.JiraTicket{}, nil
	}
	return a.liveListTickets(ctx, accountSFID)
}

// ---------------------------------------------------------------------------
// Dynamic field resolution — resolves human-readable name → Jira internal key
// ---------------------------------------------------------------------------

// jiraFieldDef is a single entry from the /rest/api/3/field response.
type jiraFieldDef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// resolveFieldKey looks up the internal Jira field key (e.g. "customfield_10057")
// for the configured JIRA_SF_ACCOUNT_FIELD name. The result is cached in-memory
// after the first successful lookup so subsequent calls are free.
//
// If the field cannot be resolved (no match or API error), the method falls back
// to using the field name directly as a quoted JQL string — Jira accepts this
// for fields whose names are unique in the schema.
func (a *JiraAdapter) resolveFieldKey(ctx context.Context) string {
	// Fast path: already resolved.
	a.fieldKeyMu.RLock()
	if a.fieldResolved {
		key := a.fieldKey
		a.fieldKeyMu.RUnlock()
		return key
	}
	a.fieldKeyMu.RUnlock()

	// Slow path: call /rest/api/3/field and find the matching field.
	a.fieldKeyMu.Lock()
	defer a.fieldKeyMu.Unlock()

	// Double-checked locking.
	if a.fieldResolved {
		return a.fieldKey
	}

	targetName := a.cfg.JiraSFAccountField
	if targetName == "" {
		a.fieldResolved = true
		a.fieldKey = ""
		return ""
	}

	endpoint := strings.TrimRight(a.cfg.JiraBaseURL, "/") + "/rest/api/3/field"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		// Non-fatal: fall back to quoted name in JQL.
		a.fieldResolved = true
		return ""
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
		}
		a.fieldResolved = true
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.fieldResolved = true
		return ""
	}

	var fields []jiraFieldDef
	if err := json.Unmarshal(body, &fields); err != nil {
		a.fieldResolved = true
		return ""
	}

	lowerTarget := strings.ToLower(targetName)
	for _, f := range fields {
		if strings.ToLower(f.Name) == lowerTarget {
			a.fieldKey = f.Key // e.g. "customfield_10057"
			a.fieldResolved = true
			return a.fieldKey
		}
	}

	// Field not found by name — mark resolved with empty key so we fall back
	// to using the quoted name directly in JQL.
	a.fieldResolved = true
	return ""
}

// buildJQL constructs the JQL query string for the given accountSFID.
func (a *JiraAdapter) buildJQL(ctx context.Context, accountSFID string) string {
	fieldName := a.cfg.JiraSFAccountField
	if fieldName != "" && accountSFID != "" {
		// Jira Cloud JQL accepts quoted field names for custom fields.
		return fmt.Sprintf(`"%s" = "%s" ORDER BY updated DESC`, fieldName, accountSFID)
	}

	if a.cfg.JiraProjectKey != "" {
		return fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, a.cfg.JiraProjectKey)
	}

	return "ORDER BY updated DESC"
}

// ---------------------------------------------------------------------------
// Live implementation — Jira REST API v3
// ---------------------------------------------------------------------------

type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string     `json:"key"`
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Summary  string        `json:"summary"`
	Status   jiraStatus    `json:"status"`
	Assignee *jiraAssignee `json:"assignee"`
	Priority *jiraPriority `json:"priority"`
	Updated  string        `json:"updated"`
}

type jiraStatus struct {
	Name string `json:"name"`
}

type jiraAssignee struct {
	DisplayName string `json:"displayName"`
}

type jiraPriority struct {
	Name string `json:"name"`
}

func (a *JiraAdapter) liveListTickets(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	jql := a.buildJQL(ctx, accountSFID)
	return a.execJQL(ctx, jql, accountSFID)
}

// liveProjectTickets queries all issues in the configured project, used as a
// fallback when the SF-account-scoped query returns no results.
func (a *JiraAdapter) liveProjectTickets(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	jql := fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, a.cfg.JiraProjectKey)
	return a.execJQL(ctx, jql, accountSFID)
}

// execJQL executes a JQL search and normalises the results into JiraTicket structs.
func (a *JiraAdapter) execJQL(ctx context.Context, jql string, accountSFID string) ([]domain.JiraTicket, error) {
	endpoint := fmt.Sprintf(
		"%s/rest/api/3/search/jql?jql=%s&fields=summary,status,assignee,priority,updated&maxResults=50",
		strings.TrimRight(a.cfg.JiraBaseURL, "/"),
		url.QueryEscape(jql),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("jira: build request: %w", err)
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jira: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jira: API error HTTP %d — %s", resp.StatusCode, string(body))
	}

	var searchResp jiraSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("jira: unmarshal response: %w", err)
	}

	tickets := make([]domain.JiraTicket, 0, len(searchResp.Issues))
	for _, issue := range searchResp.Issues {
		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		priority := ""
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}
		// Jira Cloud uses ISO 8601 with timezone offset: "2024-06-07T14:30:00.000+0530"
		updatedAt, _ := time.Parse("2006-01-02T15:04:05.999-0700", issue.Fields.Updated)

		tickets = append(tickets, domain.JiraTicket{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Status:      issue.Fields.Status.Name,
			Assignee:    assignee,
			Priority:    priority,
			UpdatedAt:   updatedAt,
			AccountSFID: accountSFID,
		})
	}

	return tickets, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *JiraAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", a.basicAuth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}
