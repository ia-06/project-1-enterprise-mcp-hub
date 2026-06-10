package seeder

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/adapters"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SeedProgress struct {
	TotalAccounts     int      `json:"totalAccounts"`
	CompletedAccounts int      `json:"completedAccounts"`
	IsComplete        bool     `json:"isComplete"`
	Logs              []string `json:"logs"`
	Mu                sync.Mutex
}

var GlobalSeedProgress = &SeedProgress{
	Logs: make([]string, 0),
}

func AddLog(msg string) {
	log.Println("[Seeder]", msg)
	GlobalSeedProgress.Mu.Lock()
	defer GlobalSeedProgress.Mu.Unlock()
	GlobalSeedProgress.Logs = append(GlobalSeedProgress.Logs, msg)
}

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

	GlobalSeedProgress.Mu.Lock()
	GlobalSeedProgress.TotalAccounts = len(accounts)
	GlobalSeedProgress.CompletedAccounts = 0
	GlobalSeedProgress.IsComplete = false
	GlobalSeedProgress.Logs = []string{}
	GlobalSeedProgress.Mu.Unlock()

	AddLog(fmt.Sprintf("START: Absolute Wipe & Seed. Found %d Salesforce Accounts.", len(accounts)))

	// 2. Postgres Wipe & Schema Update
	AddLog("START: Postgres schema update and wipe...")
	if err := s.setupPostgres(ctx); err != nil {
		AddLog(fmt.Sprintf("FAIL: Postgres setup failed: %v", err))
		return fmt.Errorf("postgres setup failed: %w", err)
	}
	AddLog("SUCCESS: Postgres tables wiped.")

	// 3. Jira Wipe
	AddLog("START: Jira wipe...")
	if err := s.jiraAdp.WipeProjectIssues(ctx); err != nil {
		AddLog(fmt.Sprintf("FAIL: Jira wipe failed: %v", err))
		return fmt.Errorf("jira wipe failed: %w", err)
	}
	AddLog("SUCCESS: Jira project issues wiped.")

	// 4. Seed Data Concurrently
	var wg sync.WaitGroup
	for _, acc := range accounts {
		wg.Add(1)
		go func(account domain.Account) {
			defer wg.Done()
			s.seedAccount(context.Background(), account)
			
			GlobalSeedProgress.Mu.Lock()
			GlobalSeedProgress.CompletedAccounts++
			GlobalSeedProgress.Mu.Unlock()
		}(acc)
	}

	wg.Wait()
	
	GlobalSeedProgress.Mu.Lock()
	GlobalSeedProgress.IsComplete = true
	GlobalSeedProgress.Mu.Unlock()
	
	AddLog("COMPLETED: All accounts seeded successfully.")
	return nil
}

func (s *SeederService) setupPostgres(ctx context.Context) error {
	_, err := s.dbPool.Exec(ctx, `ALTER TABLE customers ADD COLUMN IF NOT EXISTS mrr_cents BIGINT DEFAULT 0;`)
	if err != nil {
		return err
	}

	_, err = s.dbPool.Exec(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'unique_external_sf_id'
			) THEN
				ALTER TABLE customers ADD CONSTRAINT unique_external_sf_id UNIQUE(external_sf_id);
			END IF;
		END $$;
	`)
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
	AddLog(fmt.Sprintf("START: Seeding Account %s", acc.Name))

	// Ensure customer exists in Postgres
	var customerID string
	err := s.dbPool.QueryRow(ctx, `
		INSERT INTO customers (external_sf_id, name, industry)
		VALUES ($1, $2, $3)
		ON CONFLICT (external_sf_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id::text
	`, acc.ID, acc.Name, acc.Industry).Scan(&customerID)
	
	if err != nil {
		AddLog(fmt.Sprintf("FAIL: Upserting customer %s in Postgres: %v", acc.Name, err))
		return
	}
	AddLog(fmt.Sprintf("SUCCESS: Postgres customer %s ready", acc.Name))

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

		_, err = s.dbPool.Exec(ctx, `
			INSERT INTO sales_orders (customer_id, order_number, amount_cents, currency, status)
			VALUES ($1, $2, $3, $4, $5)
		`, customerID, fmt.Sprintf("ORD-%s-%d", acc.ID, rand.Intn(999999)), amount, "USD", status)
		if err != nil {
			AddLog(fmt.Sprintf("FAIL: Inserting order for %s: %v", acc.Name, err))
		}
	}
	AddLog(fmt.Sprintf("SUCCESS: Seeded %d orders for %s", orderCount, acc.Name))

	// Seed Jira Tickets (0 to 12)
	ticketCount := rand.Intn(13)
	for i := 0; i < ticketCount; i++ {
		desc := randomTicketDescription()
		s.jiraAdp.SeedIssue(ctx, acc.ID, desc.Title, desc.Body)
	}
	AddLog(fmt.Sprintf("SUCCESS: Seeded %d Jira tickets for %s", ticketCount, acc.Name))
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
