package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/seeder"
)

func RegisterSeedRoute(app *fiber.App, sf *adapters.SalesforceAdapter, jira *adapters.JiraAdapter, repo adapters.SalesRepository) {
	app.Post("/api/seed", func(c fiber.Ctx) error {
		// Run seeding asynchronously to avoid blocking the HTTP request
		go func() {
			s := seeder.NewSeederService(sf, jira, repo, repo.Pool())
			// Use the request context or background if we want it to survive
			// We'll use background so it completes even if client disconnects
			if err := s.WipeAndSeed(c.Context()); err != nil {
				// Just log the error, the seeder service logs anyway
			}
		}()
		
		return c.Status(202).JSON(fiber.Map{
			"status": "accepted",
			"message": "Seeding process started in the background. Please wait a few moments.",
		})
	})
	app.Get("/api/seed/status", func(c fiber.Ctx) error {
		seeder.GlobalSeedProgress.Mu.Lock()
		defer seeder.GlobalSeedProgress.Mu.Unlock()
		return c.JSON(fiber.Map{
			"totalAccounts":     seeder.GlobalSeedProgress.TotalAccounts,
			"completedAccounts": seeder.GlobalSeedProgress.CompletedAccounts,
			"isComplete":        seeder.GlobalSeedProgress.IsComplete,
			"logs":              seeder.GlobalSeedProgress.Logs,
		})
	})
}
