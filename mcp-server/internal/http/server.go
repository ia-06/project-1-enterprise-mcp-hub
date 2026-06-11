// Package http provides the Fiber application factory and middleware stack
// for the Enterprise MCP Hub backend HTTP layer.
package http

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
)

// NewServer creates and configures the Fiber application with all
// global middleware attached. It does NOT register any route handlers;
// use RegisterJSONRPCHandler and RegisterHealthRoutes for that.
func NewServer(cfg config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		// Structured error handling — never leak stack traces to clients.
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"jsonrpc": "2.0",
				"error": fiber.Map{
					"code":    -32603,
					"message": "Internal error",
					"data":    fiber.Map{"detail": err.Error()},
				},
				"id": nil,
			})
		},
		// Limit request body to 1 MB to guard against oversized payloads.
		BodyLimit: 1 * 1024 * 1024,
	})

	// ------------------------------------------------------------------
	// Middleware: Panic recovery
	// Converts any unhandled panics into a 500 JSON response instead of
	// crashing the process or leaking goroutine details.
	// ------------------------------------------------------------------
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.GoEnv != "production",
	}))

	// ------------------------------------------------------------------
	// Middleware: Request ID injection
	// Attaches X-Request-Id to every request for distributed tracing.
	// ------------------------------------------------------------------
	app.Use(requestid.New())

	// ------------------------------------------------------------------
	// Middleware: CORS
	// Allows the Next.js dev server (port 3000) to call the Go server
	// (port 8080) during local development.
	// ------------------------------------------------------------------
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "X-Request-Id", "Authorization"},
	}))

	// ------------------------------------------------------------------
	// Middleware: Structured request/response logger
	// ------------------------------------------------------------------
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)
		log.Printf(
			"[mcp-hub] method=%s path=%s status=%d duration=%s requestId=%s",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration,
			c.Locals("requestid"),
		)
		return err
	})

	return app
}
