// Package main is the entry point for the Enterprise MCP Hub backend.
// It bootstraps the Fiber HTTP server, the native MCP server (via mcp-go),
// wires all adapters, registers JSON-RPC 2.0 route handlers, and starts
// listening for incoming requests.
package main

import (
	"flag"
	"log"

	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/cache"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	internalhttp "github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/http"
	internalmcp "github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	mode := flag.String("mode", "http", "Server mode: 'http' (default) or 'stdio'")
	flag.Parse()
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
	// 4. Branch based on mode
	// ------------------------------------------------------------------
	if *mode == "stdio" {
		log.Printf("[mcp-hub] Starting MCP server in STDIO mode...")
		// Use mcp-go's battle-tested native stdio server
		if err := server.ServeStdio(mcpServer.MCPServer()); err != nil {
			log.Fatalf("MCP stdio server error: %v", err)
		}
		return
	}

	// ------------------------------------------------------------------
	// 5. Instantiate and start the Fiber HTTP application.
	// ------------------------------------------------------------------
	fiberApp := internalhttp.NewServer(cfg)
	internalhttp.RegisterJSONRPCHandler(fiberApp, cfg, mcpServer)
	internalhttp.RegisterHealthRoutes(fiberApp, cfg, salesRepo, jiraAdapter, sfAdapter)
	internalhttp.RegisterSeedRoute(fiberApp, sfAdapter, jiraAdapter, salesRepo)

	log.Printf("[mcp-hub] Starting HTTP server on %s (env=%s)", cfg.HTTPAddr, cfg.GoEnv)
	if err := fiberApp.Listen(cfg.HTTPAddr); err != nil {
		log.Fatalf("server terminated with error: %v", err)
	}
}
