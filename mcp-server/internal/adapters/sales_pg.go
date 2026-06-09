// Package adapters contains all external data-source adapters.
// This file implements the PostgreSQL-backed sales repository.
package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/config"
	"github.com/ia-06/project-1-enterprise-mcp-hub/mcp-server/internal/domain"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// SalesRepository is the contract that the HTTP and MCP layers use to access
// sales data. The Postgres implementation below satisfies this interface.
type SalesRepository interface {
	ListOrders(ctx context.Context, customerID string) ([]domain.SalesOrder, error)
	GetCustomerSummary(ctx context.Context, customerID string) (domain.SalesSummary, error)
	Ping(ctx context.Context) error
	Pool() *pgxpool.Pool
}

// ---------------------------------------------------------------------------
// PostgreSQL implementation
// ---------------------------------------------------------------------------

// pgSalesRepository implements SalesRepository using a pgxpool connection pool.
type pgSalesRepository struct {
	pool *pgxpool.Pool
}

// NewSalesRepository creates a new connection pool and verifies connectivity.
func NewSalesRepository(cfg config.Config) (SalesRepository, error) {
	pool, err := pgxpool.New(context.Background(), cfg.PgDSN)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &pgSalesRepository{pool: pool}, nil
}

// Ping checks database connectivity for health probes.
func (r *pgSalesRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Pool returns the underlying pgxpool connection pool.
func (r *pgSalesRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// ListOrders returns all sales orders, optionally filtered by customerID.
// When customerID is empty, all orders are returned (capped at 200 rows
// to guard against runaway queries on large datasets).
func (r *pgSalesRepository) ListOrders(ctx context.Context, customerID string) ([]domain.SalesOrder, error) {
	var rows pgx.Rows
	var err error

	if customerID == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, customer_id, order_number, amount_cents, currency,
			       status, closed_at, created_at
			FROM   sales_orders
			ORDER  BY created_at DESC
			LIMIT  200
		`)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT o.id, o.customer_id, o.order_number, o.amount_cents, o.currency,
			       o.status, o.closed_at, o.created_at
			FROM   sales_orders o
			JOIN   customers c ON o.customer_id = c.id
			WHERE  c.external_sf_id = $1 OR CAST(c.id AS TEXT) = $1
			ORDER  BY o.created_at DESC
		`, customerID)
	}

	if err != nil {
		return nil, fmt.Errorf("ListOrders query: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

// GetCustomerSummary returns the customer record plus aggregate totals
// for closed-won and open-pipeline orders.
func (r *pgSalesRepository) GetCustomerSummary(ctx context.Context, customerID string) (domain.SalesSummary, error) {
	// Fetch customer
	var c domain.Customer
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(external_sf_id, ''), name,
		       COALESCE(industry, ''), COALESCE(mrr_cents, 0), created_at
		FROM   customers
		WHERE  external_sf_id = $1 OR CAST(id AS TEXT) = $1
	`, customerID).Scan(&c.ID, &c.ExternalSFID, &c.Name, &c.Industry, &c.MRRCents, &c.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SalesSummary{}, fmt.Errorf("customer %q not found", customerID)
		}
		return domain.SalesSummary{}, fmt.Errorf("GetCustomerSummary customer query: %w", err)
	}

	// Aggregate totals
	var closedWon, openPipeline int64
	var orderCount int

	err = r.pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN status = 'CLOSED_WON' THEN amount_cents ELSE 0 END), 0) AS closed_won,
		  COALESCE(SUM(CASE WHEN status IN ('OPEN', 'PENDING') THEN amount_cents ELSE 0 END), 0) AS open_pipeline,
		  COUNT(*) AS order_count
		FROM sales_orders
		WHERE customer_id = $1
	`, c.ID).Scan(&closedWon, &openPipeline, &orderCount)

	if err != nil {
		return domain.SalesSummary{}, fmt.Errorf("GetCustomerSummary aggregate query: %w", err)
	}

	return domain.SalesSummary{
		Customer:            c,
		TotalClosedWonCents: closedWon,
		OpenPipelineCents:   openPipeline,
		OrderCount:          orderCount,
	}, nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

func scanOrders(rows pgx.Rows) ([]domain.SalesOrder, error) {
	orders := make([]domain.SalesOrder, 0)
	for rows.Next() {
		var o domain.SalesOrder
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.OrderNumber, &o.AmountCents,
			&o.Currency, &o.Status, &o.ClosedAt, &o.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanOrders: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ---------------------------------------------------------------------------
// Null / degraded-mode implementation
// ---------------------------------------------------------------------------

// ErrSalesUnavailable is returned by NullSalesRepository when Postgres is down.
var ErrSalesUnavailable = errors.New("sales database unavailable — Postgres not connected")

// nullSalesRepository satisfies SalesRepository with no-op methods.
// It is used when the server starts before Postgres is reachable so that
// Jira, Salesforce, and health routes still function correctly.
type nullSalesRepository struct{}

// NewNullSalesRepository returns a repository that always reports unavailable.
func NewNullSalesRepository() SalesRepository {
	return &nullSalesRepository{}
}

func (r *nullSalesRepository) Ping(_ context.Context) error {
	return ErrSalesUnavailable
}

func (r *nullSalesRepository) Pool() *pgxpool.Pool {
	return nil
}

func (r *nullSalesRepository) ListOrders(_ context.Context, _ string) ([]domain.SalesOrder, error) {
	return nil, ErrSalesUnavailable
}

func (r *nullSalesRepository) GetCustomerSummary(_ context.Context, _ string) (domain.SalesSummary, error) {
	return domain.SalesSummary{}, ErrSalesUnavailable
}
