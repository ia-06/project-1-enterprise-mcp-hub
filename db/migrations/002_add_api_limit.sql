-- =============================================================
-- Migration: 002_add_api_limit.sql
-- Adds the api_limit column to the customers table for rate limiting.
-- =============================================================

ALTER TABLE customers
ADD COLUMN IF NOT EXISTS api_limit INTEGER NOT NULL DEFAULT 1000;
