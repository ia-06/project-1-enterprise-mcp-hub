package seeder

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SeederService struct {
	sfAdp     *adapters.SalesforceAdapter
	jiraAdp   *adapters.JiraAdapter
	salesRepo adapters.SalesRepository
	dbPool    *pgxpool.Pool
}

func NewSeederService(sf *adapters.SalesforceAdapter, jira *adapters.JiraAdapter, repo adapters.SalesRepository, db *pgxpool.Pool) *SeederService {
	return &SeederService{
		sfAdp:     sf,
		jiraAdp:   jira,
		salesRepo: repo,
		dbPool:    db,
	}
}

func (s *SeederService) WipeAndSeed(ctx context.Context) error {
	log.Println("[Seeder] Starting Absolute Wipe & Seed process...")

	// 1. Fetch all SF Accounts
	accounts, err := s.sfAdp.ListAccounts(ctx, 200)
	if err != nil {
		return fmt.Errorf("failed to fetch SF accounts: %w", err)
	}

	log.Printf("[Seeder] Found %d Salesforce Accounts", len(accounts))

	// 2. Postgres Wipe & Schema Update
	if err := s.setupPostgres(ctx); err != nil {
		return fmt.Errorf("postgres setup failed: %w", err)
	}

	// 3. Jira Wipe
	if err := s.jiraAdp.WipeProjectIssues(ctx); err != nil {
		return fmt.Errorf("jira wipe failed: %w", err)
	}

	// 4. Seed Data Concurrently
	var wg sync.WaitGroup
	for _, acc := range accounts {
		wg.Add(1)
		go func(account domain.Account) {
			defer wg.Done()
			s.seedAccount(context.Background(), account)
		}(acc)
	}

	wg.Wait()
	log.Println("[Seeder] Seeding Complete!")
	return nil
}

func (s *SeederService) setupPostgres(ctx context.Context) error {
	_, err := s.dbPool.Exec(ctx, `ALTER TABLE customers ADD COLUMN IF NOT EXISTS mrr_cents BIGINT DEFAULT 0;`)
	if err != nil {
		return err
	}

	_, err = s.dbPool.Exec(ctx, `DELETE FROM sales_orders;`)
	if err != nil {
		return err
	}

	_, err = s.dbPool.Exec(ctx, `UPDATE customers SET mrr_cents = 0;`)
	return err
}

func (s *SeederService) seedAccount(ctx context.Context, acc domain.Account) {
	log.Printf("[Seeder] Seeding Account: %s", acc.Name)

	// Ensure customer exists in Postgres
	var customerID int
	err := s.dbPool.QueryRow(ctx, `
		INSERT INTO customers (external_sf_id, name, industry)
		VALUES ($1, $2, $3)
		ON CONFLICT (external_sf_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, acc.ID, acc.Name, acc.Industry).Scan(&customerID)
	
	if err != nil {
		log.Printf("[Seeder] Error upserting customer %s: %v", acc.ID, err)
		return
	}

	// Generate Random MRR ($500 to $50,000)
	mrr := rand.Intn(49500*100) + 50000
	_, _ = s.dbPool.Exec(ctx, `UPDATE customers SET mrr_cents = $1 WHERE id = $2`, mrr, customerID)

	// Seed 12-36 Orders
	orderCount := rand.Intn(25) + 12
	for i := 0; i < orderCount; i++ {
		amount := rand.Intn(10000*100) + 5000
		status := "CLOSED_WON"
		r := rand.Float32()
		if r > 0.8 {
			status = "OPEN"
		} else if r > 0.6 {
			status = "CLOSED_LOST"
		}

		_, _ = s.dbPool.Exec(ctx, `
			INSERT INTO sales_orders (customer_id, order_number, amount_cents, currency, status)
			VALUES ($1, $2, $3, $4, $5)
		`, customerID, fmt.Sprintf("ORD-%d-%d", customerID, rand.Intn(999999)), amount, "USD", status)
	}

	// Seed Jira Tickets (0 to 12)
	ticketCount := rand.Intn(13)
	for i := 0; i < ticketCount; i++ {
		desc := randomTicketDescription()
		s.jiraAdp.SeedIssue(ctx, acc.ID, desc.Title, desc.Body)
	}
}

type TicketDesc struct {
	Title string
	Body  string
}

func randomTicketDescription() TicketDesc {
	pool := []TicketDesc{
		{"Billing sync failure", "The automated billing sync is failing for this account. Needs investigation."},
		{"API Rate Limit Exceeded", "Customer is hitting the API rate limit constantly. Consider increasing tier."},
		{"Dashboard 500 Error", "When the user clicks on the reports tab, a 500 internal server error occurs."},
		{"Missing Invoice PDF", "Invoice #8892 is missing its PDF attachment in the email payload."},
		{"SSO Login Loop", "Users from this account report a SAML infinite redirect loop during login."},
	}
	return pool[rand.Intn(len(pool))]
}
