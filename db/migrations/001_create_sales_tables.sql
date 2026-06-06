-- =============================================================
-- Migration: 001_create_sales_tables.sql
-- Creates the core sales schema for the MCP Hub prototype.
-- =============================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- -------------------------------------------------------------
-- Table: customers
-- Mirrors Salesforce Account records; keyed by internal UUID
-- with an optional external_sf_id for cross-system joins.
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customers (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  external_sf_id VARCHAR(80),                          -- Salesforce Account.Id
  name           TEXT        NOT NULL,
  industry       TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for Salesforce cross-system lookups
CREATE INDEX IF NOT EXISTS idx_customers_external_sf_id ON customers(external_sf_id);

-- -------------------------------------------------------------
-- Table: sales_orders
-- Tracks individual sales orders linked to a customer.
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sales_orders (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id    UUID        NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
  order_number   TEXT        NOT NULL UNIQUE,
  amount_cents   BIGINT      NOT NULL CHECK (amount_cents >= 0),
  currency       CHAR(3)     NOT NULL DEFAULT 'USD',
  status         TEXT        NOT NULL CHECK (status IN ('OPEN', 'CLOSED_WON', 'CLOSED_LOST', 'PENDING')),
  closed_at      TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns in the MCP sales adapter
CREATE INDEX IF NOT EXISTS idx_sales_orders_customer_id ON sales_orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_status      ON sales_orders(status);
CREATE INDEX IF NOT EXISTS idx_sales_orders_created_at  ON sales_orders(created_at DESC);

-- -------------------------------------------------------------
-- Table: cache_accounts
-- Optional Supabase-compatible cache table for Salesforce
-- Account snapshots used in the Scenario A fallback path.
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cache_accounts (
  sf_id          VARCHAR(80) PRIMARY KEY,              -- Salesforce Account.Id
  name           TEXT        NOT NULL,
  tier           TEXT        NOT NULL DEFAULT 'Unknown',
  mrr_cents      BIGINT      NOT NULL DEFAULT 0,
  health_score   NUMERIC(5,2) NOT NULL DEFAULT 0,
  owner          TEXT,
  industry       TEXT,
  cached_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
