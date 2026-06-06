// Package main is the entry point for the Enterprise MCP Hub backend.
// It bootstraps the Fiber HTTP server, the native MCP server (via mcp-go),
// wires all adapters, registers JSON-RPC 2.0 route handlers, and starts
// listening for incoming requests.
package main

import (
	"log"

	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/cache"
	"github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	internalmcp "github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/mcp"
	internalhttp "github.com/your-org/project-1-enterprise-mcp-hub/mcp-server/internal/http"
)

func main() {
	// ------------------------------------------------------------------
	// 1. Load configuration from environment variables.
	// ------------------------------------------------------------------
	cfg := config.Load()

	// ------------------------------------------------------------------
	// 2. Instantiate data adapters.
	// Postgres: start in degraded mode if the DB is unavailable so that
	// the server is still reachable for health checks and Jira/Salesforce
	// data while Docker is starting up.
	// ------------------------------------------------------------------
	salesRepo, err := adapters.NewSalesRepository(cfg)
	if err != nil {
		log.Printf("[mcp-hub] WARNING: Postgres unavailable (%v). Starting in degraded mode — sales data will return errors.", err)
		salesRepo = adapters.NewNullSalesRepository()
	}

	jiraAdapter := adapters.NewJiraAdapter(cfg)

	accountCache := cache.NewSupabaseAccountCache(cfg)
	sfAdapter := adapters.NewSalesforceAdapter(cfg, accountCache)


	// ------------------------------------------------------------------
	// 3. Instantiate the native MCP server and register tools.
	// ------------------------------------------------------------------
	mcpServer := internalmcp.NewMCPServer(cfg, salesRepo, jiraAdapter, sfAdapter)

	// ------------------------------------------------------------------
	// 4. Instantiate the Fiber HTTP application.
	// ------------------------------------------------------------------
	fiberApp := internalhttp.NewServer(cfg)

	// ------------------------------------------------------------------
	// 5. Register all HTTP routes (JSON-RPC + health).
	// ------------------------------------------------------------------
	internalhttp.RegisterJSONRPCHandler(fiberApp, cfg, mcpServer)
	internalhttp.RegisterHealthRoutes(fiberApp, cfg, salesRepo, jiraAdapter, sfAdapter)

	// ------------------------------------------------------------------
	// 6. Start the HTTP server.
	// ------------------------------------------------------------------
	log.Printf("[mcp-hub] Starting server on %s (env=%s)", cfg.HTTPAddr, cfg.GoEnv)
	if err := fiberApp.Listen(cfg.HTTPAddr); err != nil {
		log.Fatalf("server terminated with error: %v", err)
	}
}
