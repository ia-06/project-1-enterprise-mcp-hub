-- =============================================================
-- Seed: sales_seed.sql
-- Inserts representative fixture data for local development.
-- Run AFTER 001_create_sales_tables.sql.
-- =============================================================

-- -------------------------------------------------------------
-- Seed: customers
-- -------------------------------------------------------------
INSERT INTO customers (id, external_sf_id, name, industry, created_at) VALUES
  ('11111111-1111-1111-1111-111111111111', '001ACME000000001', 'ACME Corporation',        'Manufacturing',    '2024-01-15 09:00:00+00'),
  ('22222222-2222-2222-2222-222222222222', '001BETA000000002', 'Beta Technologies Inc.',  'Software',         '2024-02-20 14:30:00+00'),
  ('33333333-3333-3333-3333-333333333333', '001GAMA000000003', 'Gamma Retail Group',      'Retail',           '2024-03-10 11:15:00+00'),
  ('44444444-4444-4444-4444-444444444444', '001DELT000000004', 'Delta Financial Services', 'Financial Services', '2024-04-05 08:45:00+00'),
  ('55555555-5555-5555-5555-555555555555', '001EPSI000000005', 'Epsilon Healthcare',      'Healthcare',       '2024-05-22 16:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- -------------------------------------------------------------
-- Seed: sales_orders
-- -------------------------------------------------------------
INSERT INTO sales_orders (id, customer_id, order_number, amount_cents, currency, status, closed_at, created_at) VALUES
  -- ACME Orders
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'ORD-2024-001', 125000000, 'USD', 'CLOSED_WON',  '2024-02-28 17:00:00+00', '2024-01-20 10:00:00+00'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab', '11111111-1111-1111-1111-111111111111', 'ORD-2024-002',  80000000, 'USD', 'OPEN',        NULL,                     '2024-03-01 10:00:00+00'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaac', '11111111-1111-1111-1111-111111111111', 'ORD-2024-003',  45000000, 'USD', 'CLOSED_LOST', '2024-05-10 12:00:00+00', '2024-04-01 10:00:00+00'),

  -- Beta Technologies Orders
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'ORD-2024-010',  55000000, 'USD', 'CLOSED_WON',  '2024-03-31 18:00:00+00', '2024-02-25 09:00:00+00'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbe', '22222222-2222-2222-2222-222222222222', 'ORD-2024-011',  32000000, 'USD', 'OPEN',        NULL,                     '2024-05-15 09:00:00+00'),

  -- Gamma Retail Orders
  ('cccccccc-cccc-cccc-cccc-cccccccccccc', '33333333-3333-3333-3333-333333333333', 'ORD-2024-020',  18500000, 'USD', 'CLOSED_WON',  '2024-04-20 16:00:00+00', '2024-03-15 11:00:00+00'),
  ('cccccccc-cccc-cccc-cccc-cccccccccccf', '33333333-3333-3333-3333-333333333333', 'ORD-2024-021',  22000000, 'USD', 'PENDING',     NULL,                     '2024-06-01 11:00:00+00'),

  -- Delta Financial Orders
  ('dddddddd-dddd-dddd-dddd-dddddddddddd', '44444444-4444-4444-4444-444444444444', 'ORD-2024-030', 200000000, 'USD', 'CLOSED_WON',  '2024-06-30 20:00:00+00', '2024-04-10 14:00:00+00'),

  -- Epsilon Healthcare Orders
  ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '55555555-5555-5555-5555-555555555555', 'ORD-2024-040',  75000000, 'USD', 'OPEN',        NULL,                     '2024-05-30 09:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- -------------------------------------------------------------
-- Seed: cache_accounts (Supabase Salesforce snapshot)
-- -------------------------------------------------------------
INSERT INTO cache_accounts (sf_id, name, tier, mrr_cents, health_score, owner, industry, tickets, cached_at) VALUES
  ('001ACME000000001', 'ACME Corporation',         'Enterprise',  625000, 87.5, 'Jane Smith',    'Manufacturing',      '[]'::jsonb, '2025-01-01 00:00:00+00'),
  ('001BETA000000002', 'Beta Technologies Inc.',   'Mid-Market',  145000, 72.0, 'Bob Johnson',   'Software',           '[]'::jsonb, '2025-01-01 00:00:00+00'),
  ('001GAMA000000003', 'Gamma Retail Group',       'SMB',          42000, 55.3, 'Carol Williams','Retail',             '[]'::jsonb, '2025-01-01 00:00:00+00'),
  ('001DELT000000004', 'Delta Financial Services', 'Enterprise', 1200000, 94.1, 'Alice Chen',    'Financial Services', '[]'::jsonb, '2025-01-01 00:00:00+00'),
  ('001EPSI000000005', 'Epsilon Healthcare',       'Mid-Market',  380000, 68.7, 'David Lee',     'Healthcare',         '[]'::jsonb, '2025-01-01 00:00:00+00')
ON CONFLICT (sf_id) DO NOTHING;
