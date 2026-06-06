// Package http provides the JSON-RPC 2.0 dispatcher for the Enterprise
// MCP Hub backend. All inbound POST /rpc requests are parsed, validated,
// dispatched to the appropriate MCP tool or internal handler, and returned
// as spec-compliant JSON-RPC 2.0 responses (never HTTP 500).
package http

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	internalmcp "github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/mcp"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
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

// RegisterJSONRPCHandler mounts POST /rpc on the Fiber app. It accepts both
// single JSON-RPC 2.0 objects and JSON arrays (batch requests).
func RegisterJSONRPCHandler(
	app *fiber.App,
	cfg config.Config,
	mcpServer *internalmcp.Server,
) {
	app.Post("/rpc", func(c fiber.Ctx) error {
		// Always respond with HTTP 200; JSON-RPC errors are in the body.
		c.Set("Content-Type", "application/json")

		// ----------------------------------------------------------------
		// Step 1: Parse raw body
		// ----------------------------------------------------------------
		rawBody := c.Body()
		if len(rawBody) == 0 {
			return writeRPCError(c, nil, errCodeParseError, "Parse error", nil)
		}

		// ----------------------------------------------------------------
		// Step 2: Detect single vs. batch request
		// ----------------------------------------------------------------
		var rawMsg json.RawMessage
		if err := json.Unmarshal(rawBody, &rawMsg); err != nil {
			return writeRPCError(c, nil, errCodeParseError, "Parse error", fiber.Map{
				"detail": err.Error(),
			})
		}

		requests, isBatch, err := parseRequests(rawMsg)
		if err != nil {
			return writeRPCError(c, nil, errCodeInvalidRequest, "Invalid Request", fiber.Map{
				"detail": err.Error(),
			})
		}

		// ----------------------------------------------------------------
		// Step 3: Dispatch each request
		// ----------------------------------------------------------------
		ctx, cancel := context.WithTimeout(c.Context(), cfg.RequestTimeout)
		defer cancel()

		responses := make([]rpcResponse, 0, len(requests))
		for _, req := range requests {
			resp := handleSingleRPC(ctx, mcpServer, req)
			// JSON-RPC 2.0 spec: notifications (no id) produce no response.
			if resp != nil {
				responses = append(responses, *resp)
			}
		}

		// ----------------------------------------------------------------
		// Step 4: Return single or batch response
		// ----------------------------------------------------------------
		if !isBatch && len(responses) == 1 {
			return c.JSON(responses[0])
		}
		return c.JSON(responses)
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
	case "system.healthCheck":
		result, rpcErr = handleHealthCheck(ctx, mcpServer)
	case "sales.listOrders":
		result, rpcErr = handleSalesListOrders(ctx, mcpServer, req.Params)
	case "sales.getCustomerSummary":
		result, rpcErr = handleSalesGetCustomerSummary(ctx, mcpServer, req.Params)
	case "jira.listTicketsByAccount":
		result, rpcErr = handleJiraListTickets(ctx, mcpServer, req.Params)
	case "salesforce.getAccount":
		result, rpcErr = handleSalesforceGetAccount(ctx, mcpServer, req.Params)
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
		// Distinguish between Salesforce-specific errors and generic ones.
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

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

// writeRPCError writes a spec-compliant JSON-RPC 2.0 error body with HTTP 200.
func writeRPCError(c fiber.Ctx, id interface{}, code int, msg string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(rpcResponse{
		JSONRPC: "2.0",
		Error:   &rpcError{Code: code, Message: msg, Data: data},
		ID:      id,
	})
}

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
