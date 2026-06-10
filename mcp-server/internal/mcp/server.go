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
	"sync"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// Server wraps the mcp-go MCPServer and holds references to the adapters
// so that both MCP tools and the JSON-RPC handler share the same logic.
type Server struct {
	mcpServer *server.MCPServer
	salesRepo adapters.SalesRepository
	jiraAdp   *adapters.JiraAdapter
	sfAdp     *adapters.SalesforceAdapter
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
	s.registerSystemTools()
	log.Println("[mcp] All tools registered successfully.")
}

func (s *Server) registerSalesTools() {
	// sales.listOrders
	listOrdersTool := mcpgo.NewTool("sales_listOrders",
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
	summaryTool := mcpgo.NewTool("sales_getCustomerSummary",
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
	jiraTool := mcpgo.NewTool("jira_listTicketsByAccount",
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
	sfTool := mcpgo.NewTool("salesforce_getAccount",
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
	sfListTool := mcpgo.NewTool("salesforce_listAccounts",
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

	// salesforce.searchAccounts
	sfSearchTool := mcpgo.NewTool("salesforce_searchAccounts",
		mcpgo.WithDescription("Search Salesforce Accounts by name. Returns up to 50 matching records."),
		mcpgo.WithString("query",
			mcpgo.Required(),
			mcpgo.Description("The search query string (e.g. 'Acme')."),
		),
	)
	s.mcpServer.AddTool(sfSearchTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		query := req.GetString("query", "")
		accounts, err := s.sfAdp.SearchAccounts(ctx, query)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("salesforce.searchAccounts failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"accounts": accounts})
	})
}

func (s *Server) registerSystemTools() {
	// system.customer360
	c360Tool := mcpgo.NewTool("system_customer360",
		mcpgo.WithDescription("Get a complete Customer 360 view by fetching Salesforce account, Jira tickets, and Postgres orders concurrently. Domain Dictionary: A Customer Health Score is calculated from 0-100. It is penalized heavily by open high-priority Jira tickets (e.g., SSO Loops) and boosted by high MRR. Accounts below 60 are 'Critical Risk'."),
		mcpgo.WithString("accountId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id (e.g. '001ACME000000001')."),
		),
	)
	s.mcpServer.AddTool(c360Tool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountID := req.GetString("accountId", "")
		c360, err := s.SystemCustomer360(ctx, accountID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("system.customer360 failed: %v", err)), nil
		}
		return toolResultJSON(c360)
	})

	// system.getAccountInsights
	insightsTool := mcpgo.NewTool("system_getAccountInsights",
		mcpgo.WithDescription("Analyzes ticket-to-order ratio, broken ticket percentage, and MRR, and returns a plain English string detailing why the account is failing and recommended actions."),
		mcpgo.WithString("accountId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id (e.g. '001ACME000000001')."),
		),
	)
	s.mcpServer.AddTool(insightsTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountID := req.GetString("accountId", "")
		insights, err := s.SystemGetAccountInsights(ctx, accountID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("system.getAccountInsights failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"insights": insights})
	})

	// jira.getAccountTicketTrends
	trendsTool := mcpgo.NewTool("jira_getAccountTicketTrends",
		mcpgo.WithDescription("Analyzes open tickets and summarizes the primary issue categories for a given account."),
		mcpgo.WithString("accountId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id (e.g. '001ACME000000001')."),
		),
	)
	s.mcpServer.AddTool(trendsTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountID := req.GetString("accountId", "")
		trends, err := s.JiraGetAccountTicketTrends(ctx, accountID)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("jira.getAccountTicketTrends failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"trends": trends})
	})

	// jira.escalateTicket
	escalateTool := mcpgo.NewTool("jira_escalateTicket",
		mcpgo.WithDescription("Updates a Jira issue's priority via the Atlassian REST API and adds an escalation comment."),
		mcpgo.WithString("ticketKey",
			mcpgo.Required(),
			mcpgo.Description("The Jira issue key (e.g. 'ENG-123')."),
		),
		mcpgo.WithString("newPriority",
			mcpgo.Required(),
			mcpgo.Description("The new priority level (e.g. 'Highest', 'High', 'Medium', 'Low', 'Lowest')."),
		),
	)
	s.mcpServer.AddTool(escalateTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		key := req.GetString("ticketKey", "")
		prio := req.GetString("newPriority", "")
		err := s.jiraAdp.EscalateTicket(ctx, key, prio)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("jira.escalateTicket failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"status": "success", "message": fmt.Sprintf("Ticket %s escalated to %s", key, prio)})
	})

	// system.adjustApiRateLimit
	rateLimitTool := mcpgo.NewTool("system_adjustApiRateLimit",
		mcpgo.WithDescription("Dynamically alters a client's API limits in the Postgres database via Supabase PostgREST."),
		mcpgo.WithString("accountId",
			mcpgo.Required(),
			mcpgo.Description("The Salesforce Account.Id (e.g. '001ACME000000001')."),
		),
		mcpgo.WithNumber("newLimit",
			mcpgo.Required(),
			mcpgo.Description("The new API limit (e.g. 5000)."),
		),
	)
	s.mcpServer.AddTool(rateLimitTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountID := req.GetString("accountId", "")
		limit := int(req.GetFloat("newLimit", 1000))
		err := s.salesRepo.AdjustApiRateLimit(ctx, accountID, limit)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("system.adjustApiRateLimit failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"status": "success", "message": fmt.Sprintf("API limit for account %s adjusted to %d", accountID, limit)})
	})

	// system.customer360Batch
	c360BatchTool := mcpgo.NewTool("system_customer360Batch",
		mcpgo.WithDescription("Get a complete Customer 360 view for MULTIPLE accounts simultaneously. Highly optimized bulk API call to prevent N+1 query exhaustion."),
		mcpgo.WithString("accountIds",
			mcpgo.Required(),
			mcpgo.Description("Comma-separated list of Salesforce Account IDs (e.g. '001ACME001,001ACME002')."),
		),
	)
	s.mcpServer.AddTool(c360BatchTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		accountIDsRaw := req.GetString("accountIds", "")
		accountIDs := strings.Split(accountIDsRaw, ",")
		for i := range accountIDs {
			accountIDs[i] = strings.TrimSpace(accountIDs[i])
		}

		c360Batch, err := s.SystemCustomer360Batch(ctx, accountIDs)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("system.customer360Batch failed: %v", err)), nil
		}
		return toolResultJSON(fiber_Map{"accounts": c360Batch})
	})

	// system.healthCheck
	healthTool := mcpgo.NewTool("system_healthCheck",
		mcpgo.WithDescription("Probe all backend adapter statuses (Salesforce, Jira, Postgres) to verify system resilience matrix. Note: This checks the internal infrastructure health. It does NOT reflect the health or status of the fintech client accounts themselves."),
	)
	s.mcpServer.AddTool(healthTool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		health, err := s.HealthCheck(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("system.healthCheck failed: %v", err)), nil
		}
		return toolResultJSON(health)
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

	var (
		orders  []domain.SalesOrder
		tickets []domain.JiraTicket
		wg      sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		orders, _ = s.salesRepo.ListOrders(ctx, accountID)
	}()
	go func() {
		defer wg.Done()
		tickets, _ = s.jiraAdp.ListTicketsByAccount(ctx, accountID)
	}()
	wg.Wait()

	// We need MRR from Postgres to accurately calculate health and override the Salesforce estimate.
	summary, _ := s.salesRepo.GetCustomerSummary(ctx, accountID)
	
	// Unify the deep math calculation
	enrichAccountWithDeepMetrics(acc, orders, tickets, summary)

	// Now that we have true MRR and Health Score, cache it
	s.sfAdp.CacheAccount(ctx, acc)

	return acc, nil
}

// enrichAccountWithDeepMetrics implements a robust, unified formula for account health and syncs the true Postgres MRR.
func enrichAccountWithDeepMetrics(acc *domain.Account, orders []domain.SalesOrder, tickets []domain.JiraTicket, summary domain.SalesSummary) {
	// 1. Sync True Postgres MRR
	if summary.Customer.MRRCents > 0 {
		acc.MRRCents = summary.Customer.MRRCents
	}

	// 2. Base Health
	score := 50.0

	// 3. MRR Weight (0 to +30 pts)
	// A healthy account has high MRR. We cap the bonus at $50,000 MRR (30 pts).
	mrrBonus := float64(acc.MRRCents) / 100.0 / 50000.0 * 30.0
	if mrrBonus > 30.0 {
		mrrBonus = 30.0
	}
	score += mrrBonus

	// 4. Sales Win Rate (0 to +25 pts)
	var closedWonAmount, totalAmount int64
	for _, o := range orders {
		totalAmount += o.AmountCents
		if o.Status == "CLOSED_WON" {
			closedWonAmount += o.AmountCents
		}
	}
	if totalAmount > 0 {
		winRateBonus := (float64(closedWonAmount) / float64(totalAmount)) * 25.0
		score += winRateBonus
	} else {
		score -= 10.0 // Penalty for no sales footprint at all
	}

	// 5. Order Velocity (Flat +5 pts if they have multiple orders)
	if len(orders) > 5 {
		score += 5.0
	}

	// 6. Jira Friction (-0 to -25 pts)
	// We dynamically calculate friction so a single ticket doesn't ruin the score.
	frictionPenalty := 0.0
	for _, t := range tickets {
		statusLower := strings.ToLower(t.Status)
		if statusLower == "done" || statusLower == "closed" || statusLower == "resolved" {
			continue
		}
		
		prio := strings.ToLower(t.Priority)
		switch prio {
		case "highest", "high":
			frictionPenalty += 5.0
		case "medium":
			frictionPenalty += 2.0
		default:
			frictionPenalty += 0.5
		}
	}
	// Cap the friction penalty so the account doesn't mathematically go negative just from bug reports
	if frictionPenalty > 25.0 {
		frictionPenalty = 25.0
	}
	score -= frictionPenalty

	// 7. Clamp to [0, 100]
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	acc.HealthScore = score
}

// SystemCustomer360 fetches the Salesforce account, Jira tickets, and Postgres orders concurrently.
// It computes the health score internally to avoid double-fetching.
func (s *Server) SystemCustomer360(ctx context.Context, accountID string) (fiber_Map, error) {
	var (
		acc     *domain.Account
		tickets []domain.JiraTicket
		orders  []domain.SalesOrder
		accErr  error
		wg      sync.WaitGroup
	)

	wg.Add(3)

	// 1. Fetch Salesforce Account
	go func() {
		defer wg.Done()
		acc, accErr = s.sfAdp.GetAccount(ctx, accountID)
	}()

	// 2. Fetch Jira Tickets
	go func() {
		defer wg.Done()
		tickets, _ = s.jiraAdp.ListTicketsByAccount(ctx, accountID)
	}()

	// 3. Fetch Postgres Orders
	go func() {
		defer wg.Done()
		orders, _ = s.salesRepo.ListOrders(ctx, accountID)
	}()

	wg.Wait()

	if accErr != nil {
		return nil, fmt.Errorf("salesforce unreachable: %w", accErr)
	}

	if tickets == nil {
		tickets = []domain.JiraTicket{}
	}
	if orders == nil {
		orders = []domain.SalesOrder{}
	}

	// Calculate summary and orders impact
	summary, err := s.salesRepo.GetCustomerSummary(ctx, accountID)
	if err != nil {
		summary = domain.SalesSummary{} // fallback to empty
	}

	// Use unified health score and MRR logic
	enrichAccountWithDeepMetrics(acc, orders, tickets, summary)

	// Map JiraTickets to CacheTickets
	cacheTickets := make([]domain.CacheTicket, len(tickets))
	for j, t := range tickets {
		cacheTickets[j] = domain.CacheTicket{
			Title:  t.Summary,
			Status: t.Status,
		}
	}
	acc.Tickets = cacheTickets

	// Now that we have true MRR and Health Score, cache it
	s.sfAdp.CacheAccount(ctx, acc)

	// Build the response to perfectly match Customer360 DTO of the frontend
	return fiber_Map{
		"account": acc,
		"sales": fiber_Map{
			"summary": summary,
			"orders":  orders,
		},
		"tickets": tickets,
		"meta": fiber_Map{
			"salesforceSource": acc.Source,
			"jiraMock":         true, // Dev environment
		},
	}, nil
}

// SystemCustomer360Batch fetches the Salesforce accounts, Jira tickets, and Postgres orders in bulk to avoid API exhaustion.
func (s *Server) SystemCustomer360Batch(ctx context.Context, accountIDs []string) ([]fiber_Map, error) {
	var (
		accounts  []domain.Account
		tickets   map[string][]domain.JiraTicket
		orders    map[string][]domain.SalesOrder
		summaries map[string]domain.SalesSummary
		accErr    error
		wg        sync.WaitGroup
	)

	wg.Add(3)

	// 1. Fetch Salesforce Accounts in bulk
	go func() {
		defer wg.Done()
		accounts, accErr = s.sfAdp.ListAccountsByIDs(ctx, accountIDs)
	}()

	// 2. Fetch Jira Tickets in bulk
	go func() {
		defer wg.Done()
		tickets, _ = s.jiraAdp.ListTicketsByAccounts(ctx, accountIDs)
	}()

	// 3. Fetch Postgres Orders and Summaries in bulk
	go func() {
		defer wg.Done()
		orders, _ = s.salesRepo.ListOrdersByAccounts(ctx, accountIDs)
		summaries, _ = s.salesRepo.GetCustomerSummaries(ctx, accountIDs)
	}()

	wg.Wait()

	if accErr != nil {
		return nil, fmt.Errorf("salesforce unreachable: %w", accErr)
	}

	if tickets == nil {
		tickets = make(map[string][]domain.JiraTicket)
	}
	if orders == nil {
		orders = make(map[string][]domain.SalesOrder)
	}
	if summaries == nil {
		summaries = make(map[string]domain.SalesSummary)
	}

	results := make([]fiber_Map, 0, len(accounts))

	for i := range accounts {
		acc := &accounts[i] // pointer so we can enrich
		accTickets := tickets[acc.ID]
		if accTickets == nil {
			accTickets = []domain.JiraTicket{}
		}
		accOrders := orders[acc.ID]
		if accOrders == nil {
			accOrders = []domain.SalesOrder{}
		}
		accSummary := summaries[acc.ID]

		// Use unified health score and MRR logic
		enrichAccountWithDeepMetrics(acc, accOrders, accTickets, accSummary)

		// Map JiraTickets to CacheTickets
		cacheTickets := make([]domain.CacheTicket, len(accTickets))
		for j, t := range accTickets {
			cacheTickets[j] = domain.CacheTicket{
				Title:  t.Summary,
				Status: t.Status,
			}
		}
		acc.Tickets = cacheTickets

		// Upsert deeply enriched account into cache
		s.sfAdp.CacheAccount(ctx, acc)

		results = append(results, fiber_Map{
			"account": acc,
			"sales": fiber_Map{
				"summary": accSummary,
				"orders":  accOrders,
			},
			"tickets": accTickets,
			"meta": fiber_Map{
				"salesforceSource": acc.Source,
				"jiraMock":         true, // Dev environment
			},
		})
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// Background Cache Syncer
// ---------------------------------------------------------------------------

// StartCacheSyncer launches a background goroutine that pulls live data 
// and triggers deep metric caching immediately, and then every 'interval'.
func (s *Server) StartCacheSyncer(ctx context.Context, interval time.Duration) {
	log.Printf("[mcp-hub] Cache syncer initialized, interval: %v", interval)

	syncFunc := func() {
		log.Printf("[mcp-hub] Running automated cache sync...")
		// Fetch up to 200 active accounts from Salesforce
		accounts, err := s.sfAdp.ListAccounts(ctx, 200)
		if err != nil {
			log.Printf("[mcp-hub] Cache sync error (ListAccounts): %v", err)
			return
		}

		if len(accounts) == 0 {
			log.Printf("[mcp-hub] No accounts found to sync.")
			return
		}

		accountIDs := make([]string, len(accounts))
		for i, a := range accounts {
			accountIDs[i] = a.ID
		}

		// Call batch logic which implicitly fetches Jira/Postgres,
		// enriches accounts, and upserts to cache.
		_, err = s.SystemCustomer360Batch(ctx, accountIDs)
		if err != nil {
			log.Printf("[mcp-hub] Cache sync error (Batch): %v", err)
		} else {
			log.Printf("[mcp-hub] Cache sync complete for %d accounts.", len(accountIDs))
		}
	}

	// Run immediately on startup
	go func() {
		syncFunc()

		// Then run on ticker
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[mcp-hub] Cache syncer shutting down.")
				return
			case <-ticker.C:
				syncFunc()
			}
		}
	}()
}

// SalesforceListAccounts delegates to the Salesforce adapter SOQL query.
func (s *Server) SalesforceListAccounts(ctx context.Context, limit int) ([]domain.Account, error) {
	return s.sfAdp.ListAccounts(ctx, limit)
}

// SalesforceSearchAccounts delegates to the Salesforce adapter.
func (s *Server) SalesforceSearchAccounts(ctx context.Context, query string) ([]domain.Account, error) {
	return s.sfAdp.SearchAccounts(ctx, query)
}

// SystemGetAccountInsights analyzes ticket and order data to return plain English reasoning.
func (s *Server) SystemGetAccountInsights(ctx context.Context, accountID string) (string, error) {
	summary, err := s.salesRepo.GetCustomerSummary(ctx, accountID)
	if err != nil {
		return "", err
	}
	tickets, err := s.jiraAdp.ListTicketsByAccount(ctx, accountID)
	if err != nil {
		return "", err
	}

	totalTickets := len(tickets)
	if totalTickets == 0 {
		return "Account is perfectly healthy with 0 open engineering tickets.", nil
	}

	ticketToOrderRatio := 0.0
	if summary.OrderCount > 0 {
		ticketToOrderRatio = float64(totalTickets) / float64(summary.OrderCount)
	}

	var insights strings.Builder
	insights.WriteString(fmt.Sprintf("Account has %d tickets open and %d total sales orders (Ratio: %.2f). ", totalTickets, summary.OrderCount, ticketToOrderRatio))
	
	if ticketToOrderRatio > 0.5 {
		insights.WriteString("Warning: High ticket-to-order ratio indicates heavy engineering friction relative to sales footprint. ")
	}
	if summary.Customer.MRRCents < 1000000 {
		insights.WriteString("Account MRR is relatively low, exacerbating the impact of support costs. ")
	}

	insights.WriteString("Recommended Action: Review the Jira tickets to identify systemic blockers (e.g., SSO Login Loops) and escalate them, or consider adjusting their API limits if they are being throttled.")

	return insights.String(), nil
}

// JiraGetAccountTicketTrends analyzes open tickets and summarizes the primary issue categories.
func (s *Server) JiraGetAccountTicketTrends(ctx context.Context, accountID string) (string, error) {
	tickets, err := s.jiraAdp.ListTicketsByAccount(ctx, accountID)
	if err != nil {
		return "", err
	}
	if len(tickets) == 0 {
		return "No open tickets.", nil
	}

	highPrioCount := 0
	categoryCounts := make(map[string]int)

	for _, t := range tickets {
		prio := strings.ToLower(t.Priority)
		if prio == "highest" || prio == "high" {
			highPrioCount++
		}
		
		summaryLower := strings.ToLower(t.Summary)
		if strings.Contains(summaryLower, "sso") || strings.Contains(summaryLower, "login") {
			categoryCounts["Authentication/SSO"]++
		} else if strings.Contains(summaryLower, "rate") || strings.Contains(summaryLower, "limit") {
			categoryCounts["API Rate Limits"]++
		} else if strings.Contains(summaryLower, "sync") || strings.Contains(summaryLower, "bill") {
			categoryCounts["Billing/Sync"]++
		} else {
			categoryCounts["General Bug"]++
		}
	}

	var trends strings.Builder
	trends.WriteString(fmt.Sprintf("Out of %d open tickets, %d are High/Highest priority. ", len(tickets), highPrioCount))
	
	for cat, count := range categoryCounts {
		percentage := (float64(count) / float64(len(tickets))) * 100.0
		trends.WriteString(fmt.Sprintf("%s accounts for %.1f%% of issues. ", cat, percentage))
	}

	return trends.String(), nil
}

// JiraEscalateTicket delegates to the Jira adapter.
func (s *Server) JiraEscalateTicket(ctx context.Context, ticketKey string, newPriority string) error {
	return s.jiraAdp.EscalateTicket(ctx, ticketKey, newPriority)
}

// SystemAdjustApiRateLimit delegates to the Sales repository.
func (s *Server) SystemAdjustApiRateLimit(ctx context.Context, accountID string, newLimit int) error {
	return s.salesRepo.AdjustApiRateLimit(ctx, accountID, newLimit)
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
