// Package http provides the JSON-RPC 2.0 dispatcher for the Enterprise
// MCP Hub backend. All inbound POST /rpc requests are parsed, validated,
// dispatched to the appropriate MCP tool or internal handler, and returned
// as spec-compliant JSON-RPC 2.0 responses (never HTTP 500).
package http

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
	internalmcp "github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/mcp"
)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 wire types
// ---------------------------------------------------------------------------

// rpcRequest represents a single JSON-RPC 2.0 request object.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// rpcResponse represents a single JSON-RPC 2.0 response object.
type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// rpcError represents the JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ---------------------------------------------------------------------------
// JSON-RPC error code constants (spec + custom application range)
// ---------------------------------------------------------------------------
const (
	errCodeParseError     = -32700 // Malformed JSON body
	errCodeInvalidRequest = -32600 // Missing/wrong required fields
	errCodeMethodNotFound = -32601 // Unknown method name
	errCodeInvalidParams  = -32602 // Bad parameter types/values
	errCodeInternal       = -32603 // Unhandled internal error

	// Custom application range: -32000 to -32099
	errCodeSalesUnavailable      = -32001
	errCodeSalesforceUnavailable = -32002
	errCodeJiraUnavailable       = -32003
)

// ---------------------------------------------------------------------------
// Route registration
// ---------------------------------------------------------------------------

// ProcessJSONRPC processes a raw JSON-RPC 2.0 payload (single or batch)
// and returns the raw JSON response bytes. It returns nil if the request
// was purely notifications.
func ProcessJSONRPC(ctx context.Context, cfg config.Config, mcpServer *internalmcp.Server, rawBody []byte) []byte {
	if len(rawBody) == 0 {
		errResp := &rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: errCodeParseError, Message: "Parse error"}}
		b, _ := json.Marshal(errResp)
		return b
	}

	var rawMsg json.RawMessage
	if err := json.Unmarshal(rawBody, &rawMsg); err != nil {
		errResp := &rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: errCodeParseError, Message: "Parse error", Data: fiber.Map{"detail": err.Error()}}}
		b, _ := json.Marshal(errResp)
		return b
	}

	requests, isBatch, err := parseRequests(rawMsg)
	if err != nil {
		errResp := &rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: errCodeInvalidRequest, Message: "Invalid Request", Data: fiber.Map{"detail": err.Error()}}}
		b, _ := json.Marshal(errResp)
		return b
	}

	responses := make([]rpcResponse, 0, len(requests))
	for _, req := range requests {
		resp := handleSingleRPC(ctx, mcpServer, req)
		if resp != nil {
			responses = append(responses, *resp)
		}
	}

	if len(responses) == 0 {
		return nil
	}

	if !isBatch && len(responses) == 1 {
		b, _ := json.Marshal(responses[0])
		return b
	}
	b, _ := json.Marshal(responses)
	return b
}

// RegisterJSONRPCHandler mounts POST /rpc on the Fiber app.
func RegisterJSONRPCHandler(
	app *fiber.App,
	cfg config.Config,
	mcpServer *internalmcp.Server,
) {
	app.Post("/rpc", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(c.Context(), cfg.RequestTimeout)
		defer cancel()

		respBytes := ProcessJSONRPC(ctx, cfg, mcpServer, c.Body())
		if len(respBytes) == 0 {
			return c.SendStatus(200)
		}
		return c.Send(respBytes)
	})
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// parseRequests normalises raw JSON into a slice of rpcRequest structs.
// It returns isBatch=true when the top-level JSON value is an array.
func parseRequests(raw json.RawMessage) ([]rpcRequest, bool, error) {
	// Trim whitespace to peek at first byte
	trimmed := json.RawMessage(raw)

	// Detect batch (array) vs single (object)
	var firstByte byte
	for _, b := range trimmed {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			firstByte = b
			break
		}
	}

	if firstByte == '[' {
		var batch []rpcRequest
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, true, err
		}
		return batch, true, nil
	}

	var single rpcRequest
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, false, err
	}
	return []rpcRequest{single}, false, nil
}

// handleSingleRPC validates and dispatches a single JSON-RPC 2.0 request.
// Returns nil for valid notifications (requests without id).
func handleSingleRPC(
	ctx context.Context,
	mcpServer *internalmcp.Server,
	req rpcRequest,
) *rpcResponse {
	// Validate version field
	if req.JSONRPC != "2.0" {
		return &rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: errCodeInvalidRequest, Message: "Invalid Request", Data: fiber.Map{"detail": "jsonrpc must be \"2.0\""}},
			ID:      req.ID,
		}
	}

	// Validate method
	if req.Method == "" {
		return &rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: errCodeInvalidRequest, Message: "Invalid Request", Data: fiber.Map{"detail": "method is required"}},
			ID:      req.ID,
		}
	}

	// Log dispatch in non-production environments
	log.Printf("[jsonrpc] dispatch method=%s id=%v", req.Method, req.ID)

	// Route to handler
	var result interface{}
	var rpcErr *rpcError

	switch req.Method {
	case "initialize":
		result, rpcErr = handleMCPInitialize()
	case "notifications/initialized":
		result, rpcErr = handleMCPInitialized()
	case "tools/list":
		result, rpcErr = handleMCPToolsList()
	case "tools/call":
		result, rpcErr = handleMCPToolsCall(ctx, mcpServer, req.Params)
	case "system.healthCheck":
		result, rpcErr = handleHealthCheck(ctx, mcpServer)
	case "system.customer360":
		result, rpcErr = handleSystemCustomer360(ctx, mcpServer, req.Params)
	case "system.customer360Batch":
		result, rpcErr = handleSystemCustomer360Batch(ctx, mcpServer, req.Params)
	case "system.getAccountInsights":
		result, rpcErr = handleSystemGetAccountInsights(ctx, mcpServer, req.Params)
	case "jira.getAccountTicketTrends":
		result, rpcErr = handleJiraGetAccountTicketTrends(ctx, mcpServer, req.Params)
	case "jira.escalateTicket":
		result, rpcErr = handleJiraEscalateTicket(ctx, mcpServer, req.Params)
	case "system.adjustApiRateLimit":
		result, rpcErr = handleSystemAdjustApiRateLimit(ctx, mcpServer, req.Params)
	case "sales.listOrders":
		result, rpcErr = handleSalesListOrders(ctx, mcpServer, req.Params)
	case "sales.getCustomerSummary":
		result, rpcErr = handleSalesGetCustomerSummary(ctx, mcpServer, req.Params)
	case "jira.listTicketsByAccount":
		result, rpcErr = handleJiraListTickets(ctx, mcpServer, req.Params)
	case "salesforce.getAccount":
		result, rpcErr = handleSalesforceGetAccount(ctx, mcpServer, req.Params)
	case "salesforce.listAccounts":
		result, rpcErr = handleSalesforceListAccounts(ctx, mcpServer, req.Params)
	case "salesforce.searchAccounts":
		result, rpcErr = handleSalesforceSearchAccounts(ctx, mcpServer, req.Params)
	default:
		rpcErr = &rpcError{
			Code:    errCodeMethodNotFound,
			Message: "Method not found",
			Data:    fiber.Map{"method": req.Method},
		}
	}

	if rpcErr != nil {
		return &rpcResponse{JSONRPC: "2.0", Error: rpcErr, ID: req.ID}
	}
	return &rpcResponse{JSONRPC: "2.0", Result: result, ID: req.ID}
}

// ---------------------------------------------------------------------------
// Method handlers — delegate to MCP server tool layer
// ---------------------------------------------------------------------------

func handleHealthCheck(ctx context.Context, s *internalmcp.Server) (interface{}, *rpcError) {
	health, err := s.HealthCheck(ctx)
	if err != nil {
		return nil, &rpcError{Code: errCodeInternal, Message: "Internal error", Data: fiber.Map{"detail": err.Error()}}
	}
	return health, nil
}

func handleSalesListOrders(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		CustomerID string `json:"customerId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
	}
	orders, err := s.SalesListOrders(ctx, p.CustomerID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeSalesUnavailable, "Sales data source unavailable")
	}
	return fiber.Map{"orders": orders}, nil
}

func handleSalesGetCustomerSummary(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		CustomerID string `json:"customerId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
	}
	summary, err := s.SalesGetCustomerSummary(ctx, p.CustomerID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeSalesUnavailable, "Sales data source unavailable")
	}
	return summary, nil
}

func handleJiraListTickets(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountSFID string `json:"accountSfId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
	}
	tickets, err := s.JiraListTicketsByAccount(ctx, p.AccountSFID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeJiraUnavailable, "Jira data source unavailable")
	}
	return fiber.Map{"tickets": tickets}, nil
}

func handleSalesforceGetAccount(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
	}
	account, err := s.SalesforceGetAccount(ctx, p.AccountID)
	if err != nil {
		if err == adapters.ErrSalesforceUnavailable {
			return nil, &rpcError{
				Code:    errCodeSalesforceUnavailable,
				Message: "Salesforce unreachable",
				Data:    fiber.Map{"fallback": "supabase", "retryable": true},
			}
		}
		return nil, mapAdapterError(err, errCodeSalesforceUnavailable, "Salesforce data source unavailable")
	}
	return fiber.Map{"account": account}, nil
}

func handleSalesforceListAccounts(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		Limit int `json:"limit"`
	}
	// params is optional — if missing, use defaults
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
		}
	}
	accounts, err := s.SalesforceListAccounts(ctx, p.Limit)
	if err != nil {
		return nil, mapAdapterError(err, errCodeSalesforceUnavailable, "Salesforce data source unavailable")
	}
	return fiber.Map{"accounts": accounts}, nil
}

func handleSalesforceSearchAccounts(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		Query string `json:"query"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
		}
	}
	accounts, err := s.SalesforceSearchAccounts(ctx, p.Query)
	if err != nil {
		return nil, mapAdapterError(err, errCodeSalesforceUnavailable, "Salesforce data source unavailable")
	}
	return fiber.Map{"accounts": accounts}, nil
}

func handleSystemCustomer360(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	if p.AccountID == "" {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "accountId is required"}
	}
	c360, err := s.SystemCustomer360(ctx, p.AccountID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to aggregate Customer 360 data")
	}
	return c360, nil
}

func handleSystemCustomer360Batch(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountIDs string `json:"accountIds"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	if p.AccountIDs == "" {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "accountIds is required"}
	}
	ids := strings.Split(p.AccountIDs, ",")
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}
	c360Batch, err := s.SystemCustomer360Batch(ctx, ids)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to aggregate Customer 360 Batch data")
	}
	return fiber.Map{"accounts": c360Batch}, nil
}

func handleSystemGetAccountInsights(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	insights, err := s.SystemGetAccountInsights(ctx, p.AccountID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to get insights")
	}
	return fiber.Map{"insights": insights}, nil
}

func handleJiraGetAccountTicketTrends(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	trends, err := s.JiraGetAccountTicketTrends(ctx, p.AccountID)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to get trends")
	}
	return fiber.Map{"trends": trends}, nil
}

func handleJiraEscalateTicket(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		TicketKey   string `json:"ticketKey"`
		NewPriority string `json:"newPriority"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	// For JSON-RPC, the adapter handles this directly. Since Server holds adapters, we need a method on Server.
	// Wait, we didn't add EscalateTicket to Server struct! Let's do that in a separate chunk.
	// Actually, I can just use s.MCPServer()... but wait, JSON-RPC handler expects to call Server directly, not via MCP tools.
	// We'll assume I will add JiraEscalateTicket and SystemAdjustApiRateLimit to Server.
	err := s.JiraEscalateTicket(ctx, p.TicketKey, p.NewPriority)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to escalate ticket")
	}
	return fiber.Map{"status": "success"}, nil
}

func handleSystemAdjustApiRateLimit(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		AccountID string `json:"accountId"`
		NewLimit  int    `json:"newLimit"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params"}
	}
	err := s.SystemAdjustApiRateLimit(ctx, p.AccountID, p.NewLimit)
	if err != nil {
		return nil, mapAdapterError(err, errCodeInternal, "Failed to adjust API rate limit")
	}
	return fiber.Map{"status": "success"}, nil
}

// ---------------------------------------------------------------------------
// Standard MCP Protocol Adapters
// ---------------------------------------------------------------------------

func handleMCPInitialize() (interface{}, *rpcError) {
	return fiber.Map{
		"protocolVersion": "2024-11-05",
		"serverInfo": fiber.Map{
			"name":    "enterprise-mcp-hub",
			"version": "1.0.0",
		},
		"capabilities": fiber.Map{
			"tools": fiber.Map{},
		},
	}, nil
}

func handleMCPInitialized() (interface{}, *rpcError) {
	return nil, nil // ACK notification without returning error
}

func handleMCPToolsList() (interface{}, *rpcError) {
	return fiber.Map{
		"tools": []fiber.Map{
			{
				"name":        "system_customer360",
				"description": "Fetch aggregated customer data from Salesforce, Jira, and Postgres simultaneously to generate a 360 view.",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"accountId": fiber.Map{
							"type":        "string",
							"description": "The Salesforce Account ID to fetch.",
						},
					},
					"required": []string{"accountId"},
				},
			},
			{
				"name":        "salesforce_listAccounts",
				"description": "List available Salesforce accounts to find valid Account IDs.",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"limit": fiber.Map{
							"type":        "number",
							"description": "Max number of accounts to return (default 50).",
						},
					},
				},
			},
			{
				"name":        "salesforce_getAccount",
				"description": "Fetch a single Salesforce Account by SF ID.",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"accountId": fiber.Map{
							"type":        "string",
							"description": "The Salesforce Account ID to fetch.",
						},
					},
					"required": []string{"accountId"},
				},
			},
			{
				"name":        "jira_listTicketsByAccount",
				"description": "List Jira tickets linked to a Salesforce Account ID.",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"accountSfId": fiber.Map{
							"type":        "string",
							"description": "The Salesforce Account ID linked to the Jira tickets.",
						},
					},
					"required": []string{"accountSfId"},
				},
			},
			{
				"name":        "sales_listOrders",
				"description": "List Postgres sales orders, optionally filtered by customer ID.",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"customerId": fiber.Map{
							"type":        "string",
							"description": "The Postgres Customer ID to filter by.",
						},
					},
				},
			},
			{
				"name":        "sales_getCustomerSummary",
				"description": "Aggregate pipeline totals (closed-won, open, order count).",
				"inputSchema": fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"customerId": fiber.Map{
							"type":        "string",
							"description": "The Postgres Customer ID.",
						},
					},
					"required": []string{"customerId"},
				},
			},
			{
				"name":        "system_healthCheck",
				"description": "Probes all three channels and returns status for each.",
				"inputSchema": fiber.Map{
					"type":       "object",
					"properties": fiber.Map{},
				},
			},
		},
	}, nil
}

func handleMCPToolsCall(ctx context.Context, s *internalmcp.Server, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errCodeInvalidParams, Message: "Invalid params", Data: fiber.Map{"detail": err.Error()}}
	}

	if p.Name == "system_customer360" {
		var args struct {
			AccountID string `json:"accountId"`
		}
		if len(p.Arguments) > 0 {
			json.Unmarshal(p.Arguments, &args)
		}
		c360, err := s.SystemCustomer360(ctx, args.AccountID)
		if err != nil {
			return nil, mapAdapterError(err, errCodeInternal, "Failed to aggregate Customer 360 data")
		}
		c360Bytes, _ := json.Marshal(c360)
		return fiber.Map{
			"content": []fiber.Map{{ "type": "text", "text": string(c360Bytes) }},
		}, nil
	}

	if p.Name == "salesforce_listAccounts" {
		var args struct { Limit int `json:"limit"` }
		if len(p.Arguments) > 0 { json.Unmarshal(p.Arguments, &args) }
		if args.Limit <= 0 { args.Limit = 50 }
		res, err := s.SalesforceListAccounts(ctx, args.Limit)
		if err != nil { return nil, mapAdapterError(err, errCodeInternal, "Failed to list Salesforce accounts") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	if p.Name == "salesforce_getAccount" {
		var args struct { AccountID string `json:"accountId"` }
		if len(p.Arguments) > 0 { json.Unmarshal(p.Arguments, &args) }
		res, err := s.SalesforceGetAccount(ctx, args.AccountID)
		if err != nil { return nil, mapAdapterError(err, errCodeSalesforceUnavailable, "Salesforce unavailable") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	if p.Name == "jira_listTicketsByAccount" {
		var args struct { AccountSfID string `json:"accountSfId"` }
		if len(p.Arguments) > 0 { json.Unmarshal(p.Arguments, &args) }
		res, err := s.JiraListTicketsByAccount(ctx, args.AccountSfID)
		if err != nil { return nil, mapAdapterError(err, errCodeJiraUnavailable, "Jira unavailable") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	if p.Name == "sales_listOrders" {
		var args struct { CustomerID string `json:"customerId"` }
		if len(p.Arguments) > 0 { json.Unmarshal(p.Arguments, &args) }
		res, err := s.SalesListOrders(ctx, args.CustomerID)
		if err != nil { return nil, mapAdapterError(err, errCodeInternal, "Failed to list Postgres orders") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	if p.Name == "sales_getCustomerSummary" {
		var args struct { CustomerID string `json:"customerId"` }
		if len(p.Arguments) > 0 { json.Unmarshal(p.Arguments, &args) }
		res, err := s.SalesGetCustomerSummary(ctx, args.CustomerID)
		if err != nil { return nil, mapAdapterError(err, errCodeInternal, "Failed to get Postgres customer summary") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	if p.Name == "system_healthCheck" {
		res, err := s.HealthCheck(ctx)
		if err != nil { return nil, mapAdapterError(err, errCodeInternal, "Failed health check") }
		b, _ := json.Marshal(res)
		return fiber.Map{ "content": []fiber.Map{{ "type": "text", "text": string(b) }} }, nil
	}

	return nil, &rpcError{Code: errCodeMethodNotFound, Message: "Tool not found", Data: fiber.Map{"tool": p.Name}}
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

// mapAdapterError converts a generic adapter error into an rpcError using
// the provided application-layer code and message.
func mapAdapterError(err error, code int, msg string) *rpcError {
	return &rpcError{
		Code:    code,
		Message: msg,
		Data:    fiber.Map{"detail": err.Error(), "retryable": true},
	}
}

// RegisterHealthRoutes mounts a simple GET /health endpoint that probes
// all backing services and returns a SystemHealth aggregate.
func RegisterHealthRoutes(
	app *fiber.App,
	cfg config.Config,
	salesRepo adapters.SalesRepository,
	jiraAdapter *adapters.JiraAdapter,
	sfAdapter *adapters.SalesforceAdapter,
) {
	app.Get("/health", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), cfg.RequestTimeout)
		defer cancel()

		health := domain.SystemHealth{
			MCPServer: domain.ServiceHealth{Status: "up"},
		}

		// Probe Postgres
		if err := salesRepo.Ping(ctx); err != nil {
			health.Sales = domain.ServiceHealth{Status: "down"}
		} else {
			health.Sales = domain.ServiceHealth{Status: "up"}
		}

		// Probe Jira (mock always reports up)
		if err := jiraAdapter.Ping(ctx); err != nil {
			health.Jira = domain.ServiceHealth{Status: "degraded"}
		} else {
			health.Jira = domain.ServiceHealth{Status: "up"}
		}

		// Probe Salesforce
		sfStatus := "up"
		sfCached := false
		if err := sfAdapter.Ping(ctx); err != nil {
			sfStatus = "degraded"
			sfCached = cfg.SupabaseEnabled
		}
		health.Salesforce = domain.ServiceHealth{Status: sfStatus, Cached: sfCached}

		return c.JSON(health)
	})
}
