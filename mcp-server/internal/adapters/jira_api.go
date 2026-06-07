// Package adapters — Jira Cloud REST API v3 adapter.
//
// Authentication: HTTP Basic Auth using the Atlassian account email as the
// username and a personal API token as the password. The token is generated
// at https://id.atlassian.com/manage-profile/security/api-tokens.
//
// Live mode (JIRA_USE_MOCK=false):
//   Executes a JQL search against the configured Jira Cloud instance and
//   returns normalised domain.JiraTicket values.
//
// Mock mode (JIRA_USE_MOCK=true):
//   Returns a minimal empty slice so the server stays functional during
//   local development without real credentials. All fixture arrays have
//   been permanently removed from this file.
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
	"time"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// JiraAdapter wraps the Jira Cloud REST API and normalises responses
// into domain.JiraTicket structs. It is safe for concurrent use.
type JiraAdapter struct {
	cfg    config.Config
	client *http.Client
	// basicAuth is the precomputed "Basic <base64(email:token)>" header value.
	basicAuth string
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
// In mock mode:  returns an empty slice (no fixtures).
// In live mode:  issues a JQL query that filters by the configured
//
//	custom field (JIRA_SF_ACCOUNT_FIELD) for the given accountSFID.
func (a *JiraAdapter) ListTicketsByAccount(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	if a.cfg.JiraUseMock {
		// Mock mode: no fixture data — return empty slice so UI renders gracefully.
		return []domain.JiraTicket{}, nil
	}
	return a.liveListTickets(ctx, accountSFID)
}

// ---------------------------------------------------------------------------
// Live implementation — Jira REST API v3
// ---------------------------------------------------------------------------

// jiraSearchResponse is a partial mapping of the Jira REST API v3
// /rest/api/3/search response envelope.
type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string      `json:"key"`
	Fields jiraFields  `json:"fields"`
}

type jiraFields struct {
	Summary  string         `json:"summary"`
	Status   jiraStatus     `json:"status"`
	Assignee *jiraAssignee  `json:"assignee"`
	Priority *jiraPriority  `json:"priority"`
	Updated  string         `json:"updated"`
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
	// Build a JQL query that targets the configured SF account custom field.
	// If no custom field is configured, fall back to a project-scoped query
	// so we always return something useful for testing.
	var jql string
	if a.cfg.JiraSFAccountField != "" && accountSFID != "" {
		jql = fmt.Sprintf(`"%s" = "%s" ORDER BY updated DESC`, a.cfg.JiraSFAccountField, accountSFID)
	} else if a.cfg.JiraProjectKey != "" {
		jql = fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, a.cfg.JiraProjectKey)
	} else {
		jql = "ORDER BY updated DESC"
	}

	endpoint := fmt.Sprintf(
		"%s/rest/api/3/search?jql=%s&fields=summary,status,assignee,priority,updated&maxResults=50",
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

// setHeaders applies the Jira Basic Auth and content-type headers to req.
func (a *JiraAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", a.basicAuth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

// mustParseTime is retained for any callers that may reference it from
// other files in this package. It is no longer used internally.
func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
