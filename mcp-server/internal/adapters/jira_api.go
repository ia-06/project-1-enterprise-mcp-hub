// Package adapters — Jira REST adapter.
// When JIRA_USE_MOCK=true (the $0-tier default), this adapter returns
// static fixture data that mirrors a realistic Jira JQL search response.
// When JIRA_USE_MOCK=false, it calls the real Jira REST API v3.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// JiraAdapter wraps the Jira REST API (or its mock) and normalises
// responses into domain.JiraTicket structs.
type JiraAdapter struct {
	cfg    config.Config
	client *http.Client
}

// NewJiraAdapter creates a new JiraAdapter with a pre-configured HTTP client.
func NewJiraAdapter(cfg config.Config) *JiraAdapter {
	return &JiraAdapter{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// Ping checks Jira connectivity. In mock mode it always returns nil.
func (a *JiraAdapter) Ping(_ context.Context) error {
	if a.cfg.JiraUseMock {
		return nil
	}
	// In live mode, hit the /rest/api/3/myself endpoint as a lightweight probe.
	req, err := http.NewRequest(http.MethodGet, a.cfg.JiraBaseURL+"/rest/api/3/myself", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.JiraAPIToken)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jira ping: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListTicketsByAccount returns Jira tickets associated with a Salesforce Account ID.
// In mock mode it filters static fixtures by accountSFID.
// In live mode it executes a JQL query: `cf[10001] = "<accountSFID>" ORDER BY updated DESC`.
func (a *JiraAdapter) ListTicketsByAccount(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	if a.cfg.JiraUseMock {
		return a.mockTickets(accountSFID), nil
	}
	return a.liveListTickets(ctx, accountSFID)
}

// ---------------------------------------------------------------------------
// Mock implementation
// ---------------------------------------------------------------------------

func (a *JiraAdapter) mockTickets(accountSFID string) []domain.JiraTicket {
	// Static fixture data keyed by SF Account ID suffix.
	fixturesByAccount := map[string][]domain.JiraTicket{
		"001ACME000000001": {
			{Key: "ENG-1001", Summary: "Migrate ACME data pipeline to v2 schema", Status: "In Progress", Assignee: "alice@eng.com", Priority: "P1", UpdatedAt: mustParseTime("2025-06-01T09:00:00Z"), AccountSFID: "001ACME000000001"},
			{Key: "ENG-1002", Summary: "Fix ACME SSO authentication timeout", Status: "To Do", Assignee: "bob@eng.com", Priority: "P0", UpdatedAt: mustParseTime("2025-06-03T14:30:00Z"), AccountSFID: "001ACME000000001"},
			{Key: "ENG-1003", Summary: "ACME quarterly report generation bug", Status: "Done", Assignee: "carol@eng.com", Priority: "P2", UpdatedAt: mustParseTime("2025-05-28T11:00:00Z"), AccountSFID: "001ACME000000001"},
		},
		"001BETA000000002": {
			{Key: "ENG-2001", Summary: "Beta API rate-limit integration", Status: "In Progress", Assignee: "dave@eng.com", Priority: "P1", UpdatedAt: mustParseTime("2025-06-05T08:00:00Z"), AccountSFID: "001BETA000000002"},
			{Key: "ENG-2002", Summary: "Beta webhook signature validation", Status: "To Do", Assignee: "eve@eng.com", Priority: "P2", UpdatedAt: mustParseTime("2025-06-04T17:00:00Z"), AccountSFID: "001BETA000000002"},
		},
		"001GAMA000000003": {
			{Key: "ENG-3001", Summary: "Gamma bulk import performance regression", Status: "In Progress", Assignee: "frank@eng.com", Priority: "P0", UpdatedAt: mustParseTime("2025-06-06T10:00:00Z"), AccountSFID: "001GAMA000000003"},
		},
		"001DELT000000004": {
			{Key: "ENG-4001", Summary: "Delta compliance audit log gap", Status: "To Do", Assignee: "grace@eng.com", Priority: "P0", UpdatedAt: mustParseTime("2025-06-02T12:00:00Z"), AccountSFID: "001DELT000000004"},
			{Key: "ENG-4002", Summary: "Delta MFA rollout support", Status: "Done", Assignee: "henry@eng.com", Priority: "P1", UpdatedAt: mustParseTime("2025-05-30T16:00:00Z"), AccountSFID: "001DELT000000004"},
		},
		"001EPSI000000005": {
			{Key: "ENG-5001", Summary: "Epsilon HIPAA data residency configuration", Status: "In Progress", Assignee: "iris@eng.com", Priority: "P0", UpdatedAt: mustParseTime("2025-06-07T07:30:00Z"), AccountSFID: "001EPSI000000005"},
		},
	}

	if tickets, ok := fixturesByAccount[accountSFID]; ok {
		return tickets
	}

	// Return an empty slice for unknown accounts (not an error in mock mode).
	return []domain.JiraTicket{}
}

// ---------------------------------------------------------------------------
// Live implementation
// ---------------------------------------------------------------------------

// jiraSearchResponse is a partial Jira REST API v3 search response.
type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee *struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
		Updated    string `json:"updated"`
		CustomSFID string `json:"cf_salesforce_account_id"`
	} `json:"fields"`
}

func (a *JiraAdapter) liveListTickets(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	// JQL: filter by custom field linked to Salesforce Account ID.
	jql := fmt.Sprintf(`"Salesforce Account ID" = "%s" ORDER BY updated DESC`, accountSFID)

	endpoint := fmt.Sprintf(
		"%s/rest/api/3/search?jql=%s&fields=summary,status,assignee,priority,updated",
		strings.TrimRight(a.cfg.JiraBaseURL, "/"),
		url.QueryEscape(jql),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("jira request build: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.JiraAPIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jira response read: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jira API error: HTTP %d — %s", resp.StatusCode, string(body))
	}

	var searchResp jiraSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("jira response unmarshal: %w", err)
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
		updatedAt, _ := time.Parse(time.RFC3339, issue.Fields.Updated)

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

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
