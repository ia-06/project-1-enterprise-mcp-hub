// Package domain defines the core data models (domain structs) shared
// across all adapters, MCP tools, and the JSON-RPC handler layer.
// These structs are the single source of truth for data shapes in the
// Enterprise MCP Hub backend.
package domain

import "time"

// ---------------------------------------------------------------------------
// Sales domain
// ---------------------------------------------------------------------------

// Customer mirrors a row in the `customers` table and carries an optional
// ExternalSFID for cross-system joins with Salesforce.
type Customer struct {
	ID           string    `json:"id"`
	ExternalSFID string    `json:"externalSfId,omitempty"`
	Name         string    `json:"name"`
	Industry     string    `json:"industry,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SalesOrder mirrors a row in the `sales_orders` table.
type SalesOrder struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customerId"`
	OrderNumber string     `json:"orderNumber"`
	AmountCents int64      `json:"amountCents"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// SalesSummary is the aggregate result returned by GetCustomerSummary.
type SalesSummary struct {
	Customer            Customer `json:"customer"`
	TotalClosedWonCents int64    `json:"totalClosedWonCents"`
	OpenPipelineCents   int64    `json:"openPipelineCents"`
	OrderCount          int      `json:"orderCount"`
}

// ---------------------------------------------------------------------------
// Jira domain
// ---------------------------------------------------------------------------

// JiraTicket normalises Jira's REST API response into a flat struct.
type JiraTicket struct {
	Key         string    `json:"key"`
	Summary     string    `json:"summary"`
	Status      string    `json:"status"`
	Assignee    string    `json:"assignee"`
	Priority    string    `json:"priority"`
	UpdatedAt   time.Time `json:"updatedAt"`
	AccountSFID string    `json:"accountSfId,omitempty"`
}

// ---------------------------------------------------------------------------
// Salesforce / Account domain
// ---------------------------------------------------------------------------

// Account normalises a Salesforce Account sobject (or its cache snapshot).
type Account struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Tier        string  `json:"tier"`
	MRRCents    int64   `json:"mrrCents"`
	HealthScore float64 `json:"healthScore"`
	Owner       string  `json:"owner"`
	Industry    string  `json:"industry"`
	// Source is set to "cache" when this record came from the Supabase
	// fallback cache rather than a live Salesforce API call.
	Source string `json:"source,omitempty"` // "live" | "cache"
}

// ---------------------------------------------------------------------------
// System health
// ---------------------------------------------------------------------------

// ServiceHealth captures the liveness status of a single backing service.
type ServiceHealth struct {
	Status string `json:"status"` // "up" | "degraded" | "down"
	Cached bool   `json:"cached,omitempty"`
}

// SystemHealth is the aggregate health response for the /health endpoint.
type SystemHealth struct {
	MCPServer  ServiceHealth `json:"mcpServer"`
	Sales      ServiceHealth `json:"sales"`
	Jira       ServiceHealth `json:"jira"`
	Salesforce ServiceHealth `json:"salesforce"`
}
