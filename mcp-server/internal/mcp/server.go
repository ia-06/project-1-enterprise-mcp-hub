// Package mcp wraps the mcp-go library to instantiate the native MCP server
// and expose a Server struct that the JSON-RPC handler can also call directly,
// making MCP tools available to both LLM clients (via stdio/HTTP-SSE) and the
// internal JSON-RPC 2.0 dispatch layer.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// Server wraps the mcp-go MCPServer and holds references to the adapters
// so that both MCP tools and the JSON-RPC handler share the same logic.
type Server struct {
	mcpServer  *server.MCPServer
	salesRepo  adapters.SalesRepository
	jiraAdp    *adapters.JiraAdapter
	sfAdp      *adapters.SalesforceAdapter
}

// NewMCPServer constructs the MCP server, registers all tools, and returns
// a Server that the JSON-RPC handler can also invoke directly.
func NewMCPServer(
	cfg config.Config,
	salesRepo adapters.SalesRepository,
	jiraAdp *adapters.JiraAdapter,
	sfAdp *adapters.SalesforceAdapter,
) *Server {
	s := &Server{
		salesRepo: salesRepo,
		jiraAdp:   jiraAdp,
		sfAdp:     sfAdp,
	}

	mcpSrv := server.NewMCPServer(
		"Enterprise MCP Hub",
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	s.mcpServer = mcpSrv

	s.registerTools()
	return s
}

// MCPServer exposes the underlying mcp-go server for stdio/SSE mounting.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func (s *Server) registerTools() {
	s.registerSalesTools()
	s.registerJiraTools()
	s.registerSalesforceTools()
	log.Println("[mcp] All tools registered successfully.")
}

func (s *Server) registerSalesTools() {
	// sales.listOrders
	listOrdersTool := mcpgo.NewTool("sales.listOrders",
		mcpgo.WithDescription("List all sales orders, optionally filtered by internal customer ID."),
		mcpgo.WithString("customerId",
			mcpgo.Description("Internal UUID of the customer. Omit to list all orders."),
		),
	)
	s.mcpServer.AddTool(listOrdersTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		customerID := req.GetString("customerId", "")
		orders, err := s.salesRepo.ListOrders(ctx, customerID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("sales.listOrders failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"orders": orders})
	})

	// sales.getCustomerSummary
	summaryTool := mcpgo.NewTool("sales.getCustomerSummary",
		mcpgo.WithDescription("Get aggregated sales summary (pipeline and closed-won totals) for a customer."),
		mcpgo.WithString("customerId",
			mcpgo.Required(),
			mcpgo.Description("Internal UUID of the customer."),
		),
	)
	s.mcpServer.AddTool(summaryTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		customerID := req.GetString("customerId", "")
		summary, err := s.salesRepo.GetCustomerSummary(ctx, customerID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("sales.getCustomerSummary failed: %v", err)), nil
		}
		return toolResultJSON(summary)
	})
}

func (s *Server) registerJiraTools() {
	// jira.listTicketsByAccount
	jiraTool := mcpgo.NewTool("jira.listTicketsByAccount",
		mcpgo.WithDescription("List Jira engineering tickets linked to a Salesforce Account ID."),
		mcpgo.WithString("accountSfId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id used to filter tickets."),
		),
	)
	s.mcpServer.AddTool(jiraTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountSFID := req.GetString("accountSfId", "")
		tickets, err := s.jiraAdp.ListTicketsByAccount(ctx, accountSFID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("jira.listTicketsByAccount failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"tickets": tickets})
	})
}

func (s *Server) registerSalesforceTools() {
	// salesforce.getAccount
	sfTool := mcpgo.NewTool("salesforce.getAccount",
		mcpgo.WithDescription("Retrieve a Salesforce Account by its SF Account ID. Falls back to Supabase cache if Salesforce is unreachable."),
		mcpgo.WithString("accountId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id (e.g. '001ACME000000001')."),
		),
	)
	s.mcpServer.AddTool(sfTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountID := req.GetString("accountId", "")
		account, err := s.sfAdp.GetAccount(ctx, accountID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("salesforce.getAccount failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"account": account})
	})

	// salesforce.listAccounts
	sfListTool := mcpgo.NewTool("salesforce.listAccounts",
		mcpgo.WithDescription("List Salesforce Accounts ordered by name. Used to populate the dashboard account selector."),
		mcpgo.WithNumber("limit",
			mcpgo.Description("Maximum number of accounts to return (1-200, default 50)."),
		),
	)
	s.mcpServer.AddTool(sfListTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		limit := int(req.GetFloat("limit", 50))
		accounts, err := s.sfAdp.ListAccounts(ctx, limit)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("salesforce.listAccounts failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"accounts": accounts})
	})
}

// ---------------------------------------------------------------------------
// Exposed methods — called by the JSON-RPC handler
// These methods delegate to the same adapters as the MCP tools, ensuring
// a single source of truth for all enterprise data access logic.
// ---------------------------------------------------------------------------

// HealthCheck probes all backing services and returns a SystemHealth aggregate.
func (s *Server) HealthCheck(ctx context.Context) (*domain.SystemHealth, error) {
	health := &domain.SystemHealth{
		MCPServer: domain.ServiceHealth{Status: "up"},
	}

	if err := s.salesRepo.Ping(ctx); err != nil {
		health.Sales = domain.ServiceHealth{Status: "down"}
	} else {
		health.Sales = domain.ServiceHealth{Status: "up"}
	}

	if err := s.jiraAdp.Ping(ctx); err != nil {
		health.Jira = domain.ServiceHealth{Status: "degraded"}
	} else {
		health.Jira = domain.ServiceHealth{Status: "up"}
	}

	if err := s.sfAdp.Ping(ctx); err != nil {
		health.Salesforce = domain.ServiceHealth{Status: "degraded", Cached: true}
	} else {
		health.Salesforce = domain.ServiceHealth{Status: "up"}
	}

	return health, nil
}

// SalesListOrders delegates to the sales repository.
func (s *Server) SalesListOrders(ctx context.Context, customerID string) ([]domain.SalesOrder, error) {
	return s.salesRepo.ListOrders(ctx, customerID)
}

// SalesGetCustomerSummary delegates to the sales repository.
func (s *Server) SalesGetCustomerSummary(ctx context.Context, customerID string) (domain.SalesSummary, error) {
	return s.salesRepo.GetCustomerSummary(ctx, customerID)
}

// JiraListTicketsByAccount delegates to the Jira adapter.
func (s *Server) JiraListTicketsByAccount(ctx context.Context, accountSFID string) ([]domain.JiraTicket, error) {
	return s.jiraAdp.ListTicketsByAccount(ctx, accountSFID)
}

// SalesforceGetAccount delegates to the Salesforce adapter (with cache fallback) and computes a dynamic health score.
func (s *Server) SalesforceGetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	acc, err := s.sfAdp.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Calculate dynamic health score based on Postgres orders and Jira tickets
	orders, err := s.salesRepo.ListOrders(ctx, accountID)
	if err == nil {
		tickets, err := s.jiraAdp.ListTicketsByAccount(ctx, accountID)
		if err == nil {
			score := 100.0

			// Active Jira Tickets impact: deduct based on priority
			for _, t := range tickets {
				statusLower := strings.ToLower(t.Status)
				if statusLower != "done" && statusLower != "closed" && statusLower != "resolved" {
					prio := strings.ToLower(t.Priority)
					if prio == "highest" || prio == "high" {
						score -= 15.0
					} else if prio == "medium" {
						score -= 10.0
					} else {
						score -= 5.0
					}
				}
			}

			// Sales Orders impact:
			hasClosedWon := false
			closedLostCount := 0
			for _, o := range orders {
				if o.Status == "CLOSED_WON" {
					hasClosedWon = true
				} else if o.Status == "CLOSED_LOST" {
					closedLostCount++
				}
			}

			// Deduct if the account has no sales footprint
			if len(orders) == 0 {
				score -= 10.0
			} else if !hasClosedWon {
				score -= 5.0
			}

			// Deduct for cancelled/lost deals
			score -= float64(closedLostCount) * 5.0

			if score < 0 {
				score = 0
			}
			if score > 100 {
				score = 100
			}
			acc.HealthScore = score
		}
	}

	return acc, nil
}

// SalesforceListAccounts delegates to the Salesforce adapter SOQL query.
func (s *Server) SalesforceListAccounts(ctx context.Context, limit int) ([]domain.Account, error) {
	return s.sfAdp.ListAccounts(ctx, limit)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fiber_Map is a local alias for map[string]interface{} to build response payloads.
type fiber_Map = map[string]interface{}

// toolResultJSON marshals v to JSON and wraps it in a CallToolResult text block.
func toolResultJSON(v interface{}) (*mcpgo.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return mcpgo.NewToolResultText(string(b)), nil
}
