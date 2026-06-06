<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# Act as a Principal Solutions Architect. We are initiating Phase 1 of "PROJECT 1: Enterprise Multi-Database MCP Hub". This full-stack application prototype must connect a Next.js frontend to a native Model Context Protocol (MCP) server written in Go (Fiber) that wraps three mock enterprise data layers: a PostgreSQL database (Sales data), Jira API (Engineering tickets), and Salesforce API (Customer accounts).

Using the strict rules of our \$0 free tier infrastructure framework, conduct a deep search on 2026 native MCP Go server protocols, Go Fiber JSON-RPC 2.0 handling, and Next.js App Router API configurations.

Generate and output two distinct, complete Markdown files without any conversational prose:

1. Create a "README.md" that includes:
    - Full project file tree layout.
    - Exact initialization and dependency installation commands for terminal execution.
    - Complete data schemas and REST API endpoint paths mapping Next.js to the Go backend.
    - A perfectly formatted plain-text Mermaid.js diagram illustrating data flows from User to UI, to the Go routing layer, down to the mock data systems.
    - A "System Resilience Matrix" breaking down the exact behavior, payload, and UI-state transitions for:
        * Scenario A: Salesforce CRM Unreachable (triggering a local Supabase/fallback cache state).
        * Scenario B: Invalid JSON-RPC Packet Handling (graceful validation without 500 crashes).
2. Create an "architecture_blueprint.md" that details:
    - The exact structural blueprint, function mappings, and setup rules for the Go Fiber code files.
    - The structural components, states, and client-side hooks needed for the Next.js frontend.
<!-- README.md -->

# PROJECT 1: Enterprise Multi-Database MCP Hub

Next.js App Router frontend talking to a native Go (Fiber) MCP server that wraps three mock enterprise data layers:

- PostgreSQL (Sales data)
- Jira REST API (Engineering tickets)
- Salesforce REST API (Customer accounts)

The Go server exposes both:

- Native MCP server interface (via `mcp-go`) for LLM tooling.[^1][^2][^3]
- Internal JSON-RPC 2.0 endpoint consumed by the Next.js frontend via App Router route handlers.[^4][^5][^6][^7]

All components are designed to run entirely on \$0-tier / local infrastructure (Dockerized Postgres, mock Jira/Salesforce, and optional Supabase free-tier cache).

***

## Project Structure

```text
project-1-enterprise-mcp-hub/
├─ app/                               # Next.js App Router frontend
│  ├─ layout.tsx
│  ├─ page.tsx                        # Main dashboard (Customer 360 view)
│  ├─ api/
│  │  ├─ health/route.ts              # Probes Go MCP server + data layer health
│  │  ├─ sales/
│  │  │  └─ route.ts                  # REST → JSON-RPC → Go sales (Postgres)
│  │  ├─ tickets/
│  │  │  └─ route.ts                  # REST → JSON-RPC → Go Jira
│  │  ├─ accounts/
│  │  │  └─ route.ts                  # REST → JSON-RPC → Go Salesforce
│  │  └─ customer-360/
│  │     └─ route.ts                  # Aggregates sales + tickets + accounts
│  ├─ components/
│  │  ├─ SalesSummaryCard.tsx
│  │  ├─ TicketsList.tsx
│  │  ├─ AccountProfileCard.tsx
│  │  ├─ SystemHealthBanner.tsx
│  │  └─ JsonRpcErrorToast.tsx
│  └─ lib/
│     ├─ api-client.ts                # Fetch wrapper for /api/* with JSON-RPC error mapping
│     └─ types.ts                     # Shared frontend types
│
├─ mcp-server/                        # Native Go MCP + JSON-RPC 2.0 backend
│  ├─ cmd/
│  │  └─ server/
│  │     └─ main.go                   # Fiber bootstrap + MCP server wiring
│  ├─ internal/
│  │  ├─ config/
│  │  │  └─ config.go                 # ENV loading, timeouts, base URLs
│  │  ├─ http/
│  │  │  ├─ server.go                 # Fiber app factory, middlewares
│  │  │  └─ jsonrpc_handler.go        # /rpc JSON-RPC 2.0 dispatcher
│  │  ├─ mcp/
│  │  │  ├─ server.go                 # mcp-go server instantiation, tool registration
│  │  │  └─ tools.go                  # Tool definitions for sales/jira/salesforce
│  │  ├─ adapters/
│  │  │  ├─ sales_pg.go               # PostgreSQL adapter (sales schema)
│  │  │  ├─ jira_api.go               # Jira REST adapter (mock/live)
│  │  │  └─ salesforce_api.go         # Salesforce REST adapter (mock/live)
│  │  ├─ cache/
│  │  │  └─ supabase_cache.go         # Optional Supabase-based fallback cache
│  │  └─ domain/
│  │     └─ models.go                 # Go structs for SalesOrder, JiraTicket, Account
│  └─ go.mod
│
├─ db/
│  ├─ migrations/
│  │  └─ 001_create_sales_tables.sql
│  └─ seed/
│     └─ sales_seed.sql
│
├─ infra/
│  ├─ docker-compose.yml              # Local Postgres + optional Supabase-emulator
│  └─ env.example                     # Template env vars for Go + Next.js
│
├─ README.md
└─ architecture_blueprint.md
```


***

## Prerequisites

- Node.js 20+ and pnpm or npm.
- Go 1.22+.[^8]
- Docker Desktop or any Docker-compatible runtime (for local Postgres only).
- Optional Supabase free-tier project (for Salesforce cache fallback).

***

## Initialization \& Installation Commands

### 1. Clone and bootstrap

```bash
# Clone the repo
git clone https://github.com/your-org/project-1-enterprise-mcp-hub.git
cd project-1-enterprise-mcp-hub
```

```bash
# Install Node dependencies (Next.js App Router)
pnpm install
# or
npm install
```

```bash
# Initialize Go module for MCP server
cd mcp-server
go mod init github.com/your-org/project-1-enterprise-mcp-hub/mcp-server

# Core dependencies
go get github.com/gofiber/fiber/v3
go get github.com/mark3labs/mcp-go/server
go get github.com/mark3labs/mcp-go/mcp
go get github.com/jackc/pgx/v5
go get github.com/joho/godotenv
# Optional: faster JSON encoder for Fiber
go get github.com/goccy/go-json
cd ..
```

Fiber supports swapping the JSON encoder/decoder (e.g., `goccy/go-json`) while preserving the same handler API.[^8]

### 2. Start local infrastructure (\$0 tier)

```bash
cd infra
cp env.example .env          # fill in local dev values
docker compose up -d         # starts Postgres
cd ..
```

```bash
# Run migrations + seed (using psql or any migration tool, example with psql)
psql "postgres://postgres:postgres@localhost:5432/mcp_hub?sslmode=disable" \
  -f db/migrations/001_create_sales_tables.sql

psql "postgres://postgres:postgres@localhost:5432/mcp_hub?sslmode=disable" \
  -f db/seed/sales_seed.sql
```

If you connect to Supabase’s free-tier Postgres instead of local Docker, reuse the same schema and point the Go app to Supabase via `SUPABASE_PG_URL`.

### 3. Run the Go MCP + JSON-RPC server

```bash
cd mcp-server
go run ./cmd/server
# Default: listens on :8080 for HTTP (JSON-RPC + health),
# and exposes MCP via stdio or HTTP transport depending on config.
```

The `mcp-server` uses `mcp-go` to implement an MCP server with tools that wrap Postgres, Jira, and Salesforce adapters.[^2][^3][^1]

### 4. Run the Next.js App Router frontend

```bash
cd ..
pnpm dev
# or
npm run dev
```

The frontend uses App Router `route.ts` handlers under `app/api/*` to call into the Go JSON-RPC endpoint using the native `Request` / `Response` API.[^5][^6][^7]

***

## Environment Variables

`infra/env.example` (copy to `.env` and adjust):

```bash
# Go MCP server
GO_HTTP_ADDR=":8080"
GO_ENV="development"
GO_REQUEST_TIMEOUT_MS=8000

# Postgres (local Docker or Supabase)
PG_DSN="postgres://postgres:postgres@localhost:5432/mcp_hub?sslmode=disable"

# Jira (mock/live)
JIRA_BASE_URL="https://your-domain.atlassian.net"
JIRA_API_TOKEN="local-dev-mock-token"
JIRA_USE_MOCK="true"                 # "true" = use static fixtures instead of real API

# Salesforce (mock/live)
SF_BASE_URL="https://your-instance.salesforce.com"
SF_API_TOKEN="local-dev-mock-token"
SF_USE_MOCK="true"                   # "true" = use static fixtures instead of real API

# Supabase cache (for Salesforce fallback)
SUPABASE_URL="https://your-supabase-url.supabase.co"
SUPABASE_ANON_KEY="..."
SUPABASE_ENABLED="true"

# Next.js
NEXT_PUBLIC_API_BASE_URL="http://localhost:3000"
NEXT_PUBLIC_GO_RPC_URL="http://localhost:8080/rpc"
```


***

## Data Schemas

### PostgreSQL (Sales data)

Migration: `db/migrations/001_create_sales_tables.sql`

```sql
CREATE TABLE customers (
  id            UUID PRIMARY KEY,
  external_sf_id VARCHAR(80),              -- Salesforce Account.Id
  name          TEXT NOT NULL,
  industry      TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sales_orders (
  id             UUID PRIMARY KEY,
  customer_id    UUID NOT NULL REFERENCES customers(id),
  order_number   TEXT NOT NULL UNIQUE,
  amount_cents   BIGINT NOT NULL,
  currency       CHAR(3) NOT NULL DEFAULT 'USD',
  status         TEXT NOT NULL,           -- e.g. 'OPEN', 'CLOSED_WON', 'CLOSED_LOST'
  closed_at      TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sales_orders_customer_id ON sales_orders(customer_id);
CREATE INDEX idx_sales_orders_status ON sales_orders(status);
```

Go domain models: `mcp-server/internal/domain/models.go`:

```go
type Customer struct {
  ID          string    `json:"id"`
  ExternalSFID string   `json:"externalSfId,omitempty"`
  Name        string    `json:"name"`
  Industry    string    `json:"industry,omitempty"`
  CreatedAt   time.Time `json:"createdAt"`
}

type SalesOrder struct {
  ID          string    `json:"id"`
  CustomerID  string    `json:"customerId"`
  OrderNumber string    `json:"orderNumber"`
  AmountCents int64     `json:"amountCents"`
  Currency    string    `json:"currency"`
  Status      string    `json:"status"`
  ClosedAt    *time.Time `json:"closedAt,omitempty"`
  CreatedAt   time.Time `json:"createdAt"`
}
```


### Jira (Engineering tickets)

Backed by Jira’s REST API or mock fixtures, normalized into:

```go
type JiraTicket struct {
  Key        string    `json:"key"`        // e.g. "ENG-1234"
  Summary    string    `json:"summary"`
  Status     string    `json:"status"`     // "To Do", "In Progress", "Done"
  Assignee   string    `json:"assignee"`
  Priority   string    `json:"priority"`   // "P0", "P1", etc.
  UpdatedAt  time.Time `json:"updatedAt"`
  AccountSFID string   `json:"accountSfId,omitempty"`
}
```

The adapter `jira_api.go` maps Jira’s REST format into `JiraTicket`.[^9][^10]

### Salesforce (Customer accounts)

Backed by Salesforce REST or mock fixtures:

```go
type Account struct {
  ID           string  `json:"id"`          // Salesforce Account.Id
  Name         string  `json:"name"`
  Tier         string  `json:"tier"`        // "Enterprise", "Mid-Market", etc.
  MRR          int64   `json:"mrrCents"`
  HealthScore  float64 `json:"healthScore"` // 0–100
  Owner        string  `json:"owner"`
  Industry     string  `json:"industry"`
}
```

The adapter `salesforce_api.go` wraps Salesforce REST endpoints but defaults to mock mode with static JSON when `SF_USE_MOCK="true"` so the prototype runs on \$0 tier.[^9]

***

## JSON-RPC 2.0 Schema (Go HTTP endpoint)

The Go server exposes `/rpc` which accepts JSON-RPC 2.0 requests over HTTP `POST`.[^11][^12][^4]

**Request**

```json
{
  "jsonrpc": "2.0",
  "method": "sales.getCustomerSummary",
  "params": {
    "customerId": "c123"
  },
  "id": "req-1"
}
```

**Success response**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "customer": { "id": "c123", "name": "ACME" },
    "totalClosedWonCents": 1250000,
    "openPipelineCents": 800000
  },
  "id": "req-1"
}
```

**Error response**

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Sales data source unavailable",
    "data": { "retryable": true }
  },
  "id": "req-1"
}
```

The server implements core JSON-RPC 2.0 rules: `jsonrpc` version string, `method`, `params`, and `id` in requests; `result` or `error` and matching `id` in responses.[^4]

***

## REST API: Next.js Route Handlers → Go JSON-RPC

Each Next.js route handler lives under `app/api/**/route.ts` and proxies to the Go JSON-RPC endpoint using the Web `Request`/`Response` API.[^6][^7][^5]

### `/api/health` (GET)

- Purpose: Check Go MCP server and data-layer health.
- Flow:
    - Sends JSON-RPC batch request to `/rpc` with `system.healthCheck`.
- Response example:

```json
{
  "mcpServer": { "status": "up" },
  "sales": { "status": "up" },
  "jira": { "status": "degraded" },
  "salesforce": { "status": "up", "cached": true }
}
```


### `/api/sales` (GET)

- Query params:
    - `customerId?: string`
- JSON-RPC mapping:

```json
{
  "jsonrpc": "2.0",
  "method": "sales.listOrders",
  "params": { "customerId": "c123" },
  "id": "req-1"
}
```

- Success: `200` with normalized `SalesOrder[]`.
- Error:
    - JSON-RPC error with `code = -32001` → HTTP `502` and `{ "code": "SALES_DOWN", ... }`.


### `/api/tickets` (GET)

- Query params:
    - `accountSfId: string`
- JSON-RPC mapping:

```json
{
  "jsonrpc": "2.0",
  "method": "jira.listTicketsByAccount",
  "params": { "accountSfId": "001..." },
  "id": "req-2"
}
```

- Success: `200` with `JiraTicket[]`.


### `/api/accounts` (GET)

- Query params:
    - `accountId?: string`
- JSON-RPC mapping:

```json
{
  "jsonrpc": "2.0",
  "method": "salesforce.getAccount",
  "params": { "accountId": "001..." },
  "id": "req-3"
}
```

On Salesforce unreachable:

- Go JSON-RPC returns error `code = -32002` + `data.fallback = "supabase"`.
- Next.js responds `200` with `source: "cache"` and cached account payload from Supabase.


### `/api/customer-360` (GET)

- Query params:
    - `accountId: string`
- Server-side aggregation:

1. Call `/api/accounts?accountId=...`
2. Call `/api/sales?customerId=...` (mapping SF Account to internal Customer)
3. Call `/api/tickets?accountSfId=...`

Combined response:

```json
{
  "account": { ... },
  "sales": {
    "summary": { "totalClosedWonCents": 1250000, "openPipelineCents": 800000 },
    "orders": [ ... ]
  },
  "tickets": [ ... ],
  "meta": { "salesforceSource": "live|cache", "jiraMock": true }
}
```


***

## Mermaid Diagram: End-to-End Data Flow

```mermaid
sequenceDiagram
  participant U as User
  participant UI as Next.js UI
  participant API as Next.js Route Handler (/api/*)
  participant RPC as Go Fiber JSON-RPC (/rpc)
  participant MCP as MCP Tool Layer
  participant PG as PostgreSQL (Sales)
  participant JIRA as Jira API (mock/live)
  participant SF as Salesforce API (mock/live)
  participant SB as Supabase Cache

  U->>UI: Clicks "View Customer 360"
  UI->>API: GET /api/customer-360?accountId=001...
  API->>RPC: POST /rpc (JSON-RPC 2.0 batch)
  RPC->>MCP: Dispatch methods (sales, jira, salesforce)

  MCP->>PG: SELECT orders, customers
  PG-->>MCP: Sales rows

  MCP->>JIRA: GET /rest/api/3/search?jql=...
  JIRA-->>MCP: Tickets JSON

  MCP->>SF: GET /services/data/vXX.X/sobjects/Account/001...
  alt Salesforce reachable
    SF-->>MCP: Account JSON
  else Salesforce unreachable (Scenario A)
    SF--xMCP: Timeout / 5xx
    MCP->>SB: SELECT FROM cache.accounts WHERE id = 001...
    SB-->>MCP: Cached Account JSON
  end

  MCP-->>RPC: Aggregated tool results
  RPC-->>API: JSON-RPC 2.0 response
  API-->>UI: REST JSON response (Customer 360 DTO)
  UI-->>U: Render cards, lists, and SystemHealth banner
```


***

## System Resilience Matrix

System behavior for specific failure scenarios, including Go JSON-RPC payloads, Next.js responses, and UI state transitions. JSON-RPC error codes follow custom `-32000` to `-32099` application range on top of JSON-RPC 2.0.[^4]

### Scenario A: Salesforce CRM Unreachable → Supabase Fallback Cache

| Dimension | Behavior |
| :-- | :-- |
| **Trigger** | Network timeout or `5xx` from Salesforce REST API for `salesforce.getAccount` / `salesforce.listAccounts` |
| **Go adapter behavior** | `salesforce_api.go` wraps HTTP client with context deadline (`GO_REQUEST_TIMEOUT_MS`) and error mapping |
| **JSON-RPC error** | `error.code = -32002`, `error.message = "Salesforce unreachable"`, `error.data = { "fallback": "supabase" }` |
| **Cache lookup** | `supabase_cache.go` queries `cache.accounts` by Account.Id when `SUPABASE_ENABLED="true"` |
| **JSON-RPC success (fallback)** | When cache hit: Go server returns **success** `result` containing cached Account and `result.meta.source = "cache"` |
| **JSON-RPC failure (no cache)** | When cache miss: error remains error; Go sets `data.retryable = true` |
| **Next.js /api/accounts response** | - Cache hit: HTTP `200`, `{ account, source: "cache", resiliency: { salesforce: "unreachable" } }`<br>- Cache miss: HTTP `502`, `{ code: "SALESFORCE_DOWN", retryable: true }` |
| **UI state transition (hit)** | - `AccountProfileCard` renders as normal but with a "Cached snapshot" badge.<br>- `SystemHealthBanner` shows yellow warning "Salesforce degraded, serving cached data." |
| **UI state transition (miss)** | - `AccountProfileCard` shows skeleton + inline error: "Live CRM unreachable; retry later."<br>- Dashboard remains usable; sales + tickets sections still render. |
| **Logging \& alerts** | - Fiber middleware logs structured error with `"source":"salesforce"` and HTTP status.<br>- Optional hook to send alert to logging sink only on first failure within rolling window. |

### Scenario B: Invalid JSON-RPC Packet Handling

| Dimension | Behavior |
| :-- | :-- |
| **Trigger** | Malformed request body, missing `jsonrpc`, unknown `method`, non-object `params`, or mismatched `id` |
| **Go JSON parsing** | `json.Unmarshal` or `goccy/go-json` fails; `jsonrpc_handler.go` catches and maps to JSON-RPC parse/invalid errors. [^8] |
| **JSON-RPC error (parse)** | For malformed JSON → `code = -32700`, `message = "Parse error"` |
| **JSON-RPC error (invalid)** | For invalid structure → `code = -32600`, `message = "Invalid Request"` |
| **JSON-RPC error (method)** | For unknown `method` → `code = -32601`, `message = "Method not found"` |
| **HTTP status** | Always HTTP `200` with JSON-RPC error body, never `500`, to comply with JSON-RPC 2.0 expectations over HTTP. [^4] |
| **Next.js /api/* behavior** | Route handler inspects JSON-RPC `error.code` and maps:<br>- `-32700` / `-32600` → HTTP `400` and `{ code: "INVALID_RPC" }`<br>- `-32601` → HTTP `404` and `{ code: "RPC_METHOD_NOT_FOUND" }` |
| **UI state transition** | - `JsonRpcErrorToast` shows user-friendly messages like "Temporary issue loading data. Please retry."<br>- Component-level fallback UI uses safe default empty states, not crashes. |
| **Developer feedback** | Error logs include `rpc_method`, `rpc_id`, `error_code`, plus raw request in debug mode only. |

The JSON-RPC mapping adheres to the official JSON-RPC 2.0 error semantics for parse and invalid request handling, ensuring protocol-level correctness while surfacing stable HTTP responses.[^4]

***

<!-- architecture_blueprint.md -->

# Architecture Blueprint: Enterprise Multi-Database MCP Hub

This blueprint specifies the exact structural layout, function mappings, and setup rules for:

- Go Fiber + MCP server (`mcp-server/`)
- Next.js App Router frontend (`app/`)

The Go server uses `mcp-go` for native MCP semantics and Fiber for HTTP, while the Next.js frontend uses App Router `route.ts` handlers to call JSON-RPC 2.0 endpoints.[^7][^13][^1][^5][^6]

***

## Go MCP Server: Structural Blueprint

### Directory Layout (Back-End)

```text
mcp-server/
├─ cmd/
│  └─ server/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ http/
│  │  ├─ server.go
│  │  └─ jsonrpc_handler.go
│  ├─ mcp/
│  │  ├─ server.go
│  │  └─ tools.go
│  ├─ adapters/
│  │  ├─ sales_pg.go
│  │  ├─ jira_api.go
│  │  └─ salesforce_api.go
│  ├─ cache/
│  │  └─ supabase_cache.go
│  └─ domain/
│     └─ models.go
└─ go.mod
```


***

### `cmd/server/main.go`

Responsibilities:

- Load config via `config.Load()`.
- Instantiate Fiber app using `http.NewServer(cfg)`.
- Instantiate MCP server via `mcp.NewMCPServer(cfg)`.
- Wire MCP tools with adapters.
- Mount `/rpc` JSON-RPC handler and `/health` routes.
- Start HTTP server on `cfg.HTTPAddr`.

Pseudocode:

```go
func main() {
  cfg := config.Load()

  fiberApp := http.NewServer(cfg)
  mcpServer := mcp.NewMCPServer(cfg)

  http.RegisterJSONRPCHandlers(fiberApp, cfg, mcpServer)
  http.RegisterHealthRoutes(fiberApp, cfg)

  if err := fiberApp.Listen(cfg.HTTPAddr); err != nil {
    log.Fatal(err)
  }
}
```


***

### `internal/config/config.go`

Responsibilities:

- Read env variables (`GO_HTTP_ADDR`, `PG_DSN`, `JIRA_*`, `SF_*`, `SUPABASE_*`).
- Derive timeouts and circuit-breaker thresholds.

Key fields:

```go
type Config struct {
  HTTPAddr          string
  RequestTimeout    time.Duration
  PgDSN             string
  JiraBaseURL       string
  JiraAPIToken      string
  JiraUseMock       bool
  SfBaseURL         string
  SfAPIToken        string
  SfUseMock         bool
  SupabaseURL       string
  SupabaseAnonKey   string
  SupabaseEnabled   bool
}
```


***

### `internal/http/server.go`

Responsibilities:

- Create `fiber.App` with JSON encoder/decoder and recover middleware.
- Attach global middleware:
    - Request logging (structured).
    - Panic recovery to `500` with JSON.
    - Request timeout context.
- Provide helper for sending JSON-RPC responses.

Key patterns:

- Use Fiber’s configurable JSON encoder/decoder to potentially swap in `goccy/go-json` for speed.[^8]
- Apply `Recover`-like middleware equivalent (either custom or Fiber’s built-in) to avoid panics leaking as crashes.

Example:

```go
func NewServer(cfg config.Config) *fiber.App {
  app := fiber.New(fiber.Config{
    JSONEncoder: json.Marshal,
    JSONDecoder: json.Unmarshal,
  })

  app.Use(func(c fiber.Ctx) error {
    // request ID, logging context, timeout injection
    return c.Next()
  })

  app.Use(func(c fiber.Ctx) error {
    defer func() {
      if r := recover(); r != nil {
        _ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
          "error": "internal_server_error",
        })
      }
    }()
    return c.Next()
  })

  return app
}
```


***

### `internal/http/jsonrpc_handler.go`

Responsibilities:

- Implement JSON-RPC 2.0 over HTTP `POST /rpc`.[^12][^11][^4]
- Decode single or batch requests.
- Validate required fields:
    - `jsonrpc == "2.0"`
    - Non-empty `method`
    - `id` present (unless notification semantics are added later).
- Route methods to MCP tools or internal handlers.

Core flow:

```go
func RegisterJSONRPCHandlers(app *fiber.App, cfg config.Config, mcpServer *mcp.Server) {
  app.Post("/rpc", func(c fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.Context(), cfg.RequestTimeout)
    defer cancel()

    // Decode body
    var req json.RawMessage
    if err := json.Unmarshal(c.Body(), &req); err != nil {
      return writeRPCError(c, nil, -32700, "Parse error", nil)
    }

    // Detect single vs batch, then normalize
    requests, isBatch, err := parseRequests(req)
    if err != nil {
      return writeRPCError(c, nil, -32600, "Invalid Request", nil)
    }

    responses := make([]rpcResponse, 0, len(requests))
    for _, r := range requests {
      resp := handleSingleRPC(ctx, mcpServer, r)
      if resp != nil {
        responses = append(responses, *resp)
      }
    }

    if !isBatch && len(responses) == 1 {
      return c.JSON(responses[^0])
    }
    return c.JSON(responses)
  })
}
```

`handleSingleRPC` maps JSON-RPC `method` names to:

- `system.healthCheck`
- `sales.listOrders`
- `sales.getCustomerSummary`
- `jira.listTicketsByAccount`
- `salesforce.getAccount`

Each handler uses MCP tools or adapters underneath.

Error mapping:

- JSON parse failure → `code = -32700`.
- Invalid request structure → `code = -32600`.
- Unknown method → `code = -32601`.
- Application errors → custom `-32000`..`-32099`.[^4]

***

### `internal/mcp/server.go`

Responsibilities:

- Instantiate MCP server using `mcp-go`.[^3][^1][^2]
- Register tools for each domain.
- Expose MCP over:
    - stdio (for LLM clients)
    - HTTP/SSE if needed (optional for this phase).

Example pattern:

```go
func NewMCPServer(cfg config.Config) *server.MCPServer {
  s := server.NewMCPServer(
    "Enterprise MCP Hub",
    "0.1.0",
    server.WithToolCapabilities(true),
  )

  // Tools registered in tools.go
  registerSalesTools(s)
  registerJiraTools(s)
  registerSalesforceTools(s)

  return s
}
```

MCP tools correspond to JSON-RPC `method` names, enabling both LLM and frontend reuse.[^13][^1][^9]

***

### `internal/mcp/tools.go`

Responsibility:

- Define and register MCP tools for:
    - Sales: `sales.listOrders`, `sales.getCustomerSummary`.
    - Jira: `jira.listTicketsByAccount`.
    - Salesforce: `salesforce.getAccount`.

Each tool:

- Specifies input parameters (e.g., `accountId`, `customerId`).
- Uses adapters to fetch data.
- Returns JSON-serializable results.

Example (pseudocode):

```go
func registerSalesTools(s *server.MCPServer) {
  tool := mcp.NewTool("sales.getCustomerSummary",
    mcp.WithDescription("Get sales summary for a customer"),
    mcp.WithString("customerId", mcp.Required()),
  )

  s.AddTool(tool, func(ctx context.Context, args mcp.ToolArgs) (any, error) {
    customerID := args.String("customerId")
    // call sales_pg adapter
    return summary, nil
  })
}
```


***

### `internal/adapters/sales_pg.go`

Responsibilities:

- Manage Postgres connection pool via `pgxpool`.
- Provide typed methods:

```go
type SalesRepository interface {
  ListOrders(ctx context.Context, customerID string) ([]domain.SalesOrder, error)
  GetCustomerSummary(ctx context.Context, customerID string) (domain.SalesSummary, error)
}
```

`ListOrders` executes SQL with prepared statements; `GetCustomerSummary` executes aggregate queries.

***

### `internal/adapters/jira_api.go`

Responsibilities:

- Call Jira REST API when `JIRA_USE_MOCK=false`.[^10][^9]
- Otherwise read from local JSON fixtures under `testdata/jira/*.json`.
- Map responses to `domain.JiraTicket`.

***

### `internal/adapters/salesforce_api.go`

Responsibilities:

- Call Salesforce REST API when `SF_USE_MOCK=false`.
- When unreachable, set sentinel error `ErrSalesforceUnavailable` which the caller maps to JSON-RPC error `-32002`.
- In fallback path, call `cache.AccountCache` to read Supabase snapshot.

***

### `internal/cache/supabase_cache.go`

Responsibilities:

- Thin Supabase client using HTTP + API key.
- Read-only methods for Phase 1:

```go
type AccountCache interface {
  GetAccount(ctx context.Context, id string) (*domain.Account, error)
}
```

If `SUPABASE_ENABLED=false`, returns `ErrCacheDisabled` and the caller falls back to direct error.

***

## Next.js Frontend: Structural Components \& Client Hooks

### App Directory Layout

```text
app/
├─ layout.tsx
├─ page.tsx
├─ api/
│  ├─ health/route.ts
│  ├─ sales/route.ts
│  ├─ tickets/route.ts
│  ├─ accounts/route.ts
│  └─ customer-360/route.ts
├─ components/
│  ├─ SalesSummaryCard.tsx
│  ├─ TicketsList.tsx
│  ├─ AccountProfileCard.tsx
│  ├─ SystemHealthBanner.tsx
│  └─ JsonRpcErrorToast.tsx
└─ lib/
   ├─ api-client.ts
   └─ types.ts
```

Route handlers use the App Router `route.ts` convention with exported HTTP methods.[^5][^6][^7]

***

### Shared Types: `app/lib/types.ts`

Defines DTOs used between route handlers and React components:

```ts
export type Account = {
  id: string;
  name: string;
  tier: string;
  mrrCents: number;
  healthScore: number;
  owner: string;
  industry: string;
  source?: "live" | "cache";
};

export type SalesOrder = {
  id: string;
  customerId: string;
  orderNumber: string;
  amountCents: number;
  currency: string;
  status: string;
  closedAt?: string;
  createdAt: string;
};

export type JiraTicket = {
  key: string;
  summary: string;
  status: string;
  assignee: string;
  priority: string;
  updatedAt: string;
};

export type Customer360 = {
  account: Account | null;
  sales: {
    summary: {
      totalClosedWonCents: number;
      openPipelineCents: number;
    };
    orders: SalesOrder[];
  };
  tickets: JiraTicket[];
  meta: {
    salesforceSource: "live" | "cache" | "none";
  };
};

export type SystemHealth = {
  mcpServer: { status: "up" | "down" };
  sales: { status: "up" | "down" };
  jira: { status: "up" | "degraded" | "down" };
  salesforce: { status: "up" | "degraded" | "down"; cached: boolean };
};
```


***

### API Client: `app/lib/api-client.ts`

Responsibilities:

- Wrap `fetch` calls to `/api/*`.
- Normalize error handling from JSON-RPC mapping.

Example:

```ts
async function handleResponse<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => null);

  if (!res.ok) {
    const code = body?.code ?? "UNKNOWN_ERROR";
    throw new Error(code);
  }
  return body as T;
}

export async function getCustomer360(accountId: string): Promise<Customer360> {
  const res = await fetch(`/api/customer-360?accountId=${encodeURIComponent(accountId)}`, {
    cache: "no-store",
  });
  return handleResponse<Customer360>(res);
}
```


***

### Route Handlers: `app/api/*/route.ts`

All route handlers:

- Validate query parameters.
- Build JSON-RPC request body.
- `fetch` Go server at `NEXT_PUBLIC_GO_RPC_URL` (`POST /rpc`).
- Map JSON-RPC errors to HTTP status and error codes.

Example `app/api/accounts/route.ts`:

```ts
import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const accountId = req.nextUrl.searchParams.get("accountId");
  if (!accountId) {
    return NextResponse.json({ code: "MISSING_ACCOUNT_ID" }, { status: 400 });
  }

  const rpcReq = {
    jsonrpc: "2.0",
    method: "salesforce.getAccount",
    params: { accountId },
    id: "accounts-" + Date.now(),
  };

  const rpcRes = await fetch(process.env.NEXT_PUBLIC_GO_RPC_URL!, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rpcReq),
  });

  const body = await rpcRes.json();

  if (body.error) {
    if (body.error.code === -32002 && body.error.data?.fallback === "supabase" && body.error.data?.cache) {
      // In full implementation this branch would be hit only if cache miss; for now treat as 502.
    }
    const status =
      body.error.code === -32700 || body.error.code === -32600 ? 400 :
      body.error.code === -32601 ? 404 :
      502;

    return NextResponse.json({ code: "SALESFORCE_ERROR", rpc: body.error }, { status });
  }

  const account = body.result.account;
  return NextResponse.json(account, { status: 200 });
}
```

Route handler structure follows the Next.js App Router documentation for custom `route.ts` with `GET` exports.[^6][^7][^5]

***

### Top-Level Page: `app/page.tsx`

Responsibilities:

- Client-side state holding:
    - Selected `accountId`.
    - `Customer360` result.
    - `SystemHealth` result.
    - Loading and error states.
- On mount:
    - Call `/api/health` to populate `SystemHealth`.
- On account selection:
    - Trigger `/api/customer-360`.

Key hooks:

```ts
"use client";

export default function HomePage() {
  const [accountId, setAccountId] = useState<string | null>(null);
  const [data, setData] = useState<Customer360 | null>(null);
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorCode, setErrorCode] = useState<string | null>(null);

  useEffect(() => {
    async function loadHealth() {
      try {
        const res = await fetch("/api/health", { cache: "no-store" });
        if (!res.ok) return;
        setHealth(await res.json());
      } catch {
        // Silent health failure: banner will show "unknown"
      }
    }
    loadHealth();
  }, []);

  const handleLoad = async () => {
    if (!accountId) return;
    setLoading(true);
    setErrorCode(null);
    try {
      const res = await fetch(`/api/customer-360?accountId=${encodeURIComponent(accountId)}`, {
        cache: "no-store",
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setErrorCode(body?.code ?? "UNKNOWN");
        return;
      }
      const body = (await res.json()) as Customer360;
      setData(body);
    } finally {
      setLoading(false);
    }
  };

  // Render layout with SystemHealthBanner, cards, etc.
}
```


***

### UI Components \& State Mapping

#### `SystemHealthBanner.tsx`

Props:

```ts
type Props = { health: SystemHealth | null };
```

Behavior:

- When `health.salesforce.status === "degraded" && health.salesforce.cached`, display warning “Salesforce degraded, serving cached data.”
- When any service `status === "down"`, display error banner.
- When `health == null`, show neutral “System health unknown”.


#### `AccountProfileCard.tsx`

Props:

```ts
type Props = { account: Account | null };
```

Behavior:

- When `account === null`, show placeholder.
- When `account.source === "cache"`, show badge “Cached snapshot”.


#### `SalesSummaryCard.tsx`

Props:

```ts
type Props = { summary: Customer360["sales"]["summary"] | null };
```

Behavior:

- Renders totals with currency formatting.
- When no data, show “No sales data”.


#### `TicketsList.tsx`

Props:

```ts
type Props = { tickets: JiraTicket[] };
```

Behavior:

- When `tickets.length === 0`, show “No active tickets”.
- When Jira is degraded but not down, show subtle warning icon.


#### `JsonRpcErrorToast.tsx`

Props:

```ts
type Props = { errorCode: string | null; onClose: () => void };
```

Behavior:

- Map error codes from route handlers to user-friendly messages:
    - `INVALID_RPC` → “Temporary issue decoding data.”
    - `SALESFORCE_DOWN` → “CRM temporarily unavailable; retry later.”
- Auto-dismiss after a timeout or manual close.

***

### System Resilience Behavior (Front-End)

- **Salesforce unreachable (Scenario A)**:
    - If backend responds with cached Account (`source: "cache"`), UI shows normal dashboard with “Cached snapshot” badge and a yellow SystemHealth banner as described in the README.
    - If backend returns `502` with `code: "SALESFORCE_DOWN"`, main dashboard still renders sales and tickets; Account section shows error state but the rest of the page remains interactive.
- **Invalid JSON-RPC packet (Scenario B)**:
    - Frontend never sees a raw `500`; route handlers map parse/invalid/method errors to `400` or `404` with `code: "INVALID_RPC"` or `code: "RPC_METHOD_NOT_FOUND"`.
    - UI displays `JsonRpcErrorToast` and the affected widgets show empty or last-known-good state, avoiding full-page crashes.

***

### App Router \& MCP Integration Considerations

- Route handlers under `app/api/*/route.ts` are the only surface exposed to the browser; they shield the client from JSON-RPC details and MCP tool shapes.[^7][^5][^6]
- MCP server can also be used directly by LLM tools (e.g., via `.mcp.json` pointing at the MCP server HTTP or stdio endpoint) following MCP integration guidance.[^1][^13][^9]
- The same MCP tools used by the JSON-RPC handler can be reused by LLM clients without introducing another backend, ensuring a single source of truth for enterprise data access logic.[^2][^3][^13][^1]
<span style="display:none">[^14][^15]</span>

<div align="center">⁂</div>

[^1]: https://github.com/mark3labs/mcp-go

[^2]: https://dev.to/eminetto/creating-an-mcp-server-using-go-3foe

[^3]: https://navendu.me/posts/mcp-server-go/

[^4]: https://www.jsonrpc.org/archive_json-rpc.org/specification.html

[^5]: https://en.nextjs.im/docs/app/building-your-application/routing/route-handlers

[^6]: https://nextjs.org/docs/app/getting-started/route-handlers

[^7]: https://nextjs.org/docs/app/api-reference/file-conventions/route

[^8]: https://docs.gofiber.io/guide/faster-fiber/

[^9]: https://www.augmentcode.com/mcp/mcp-golang

[^10]: https://github.com/gofiber/websocket/issues/141

[^11]: https://www.reddit.com/r/golang/comments/1dfp8zv/a_library_for_creating_jsonrpc_20_servers_in_go/

[^12]: https://pkg.go.dev/net/rpc/jsonrpc

[^13]: https://nextjs.org/docs/app/guides/mcp

[^14]: https://github.com/vertile-ai/next-mcp-server/blob/main/README.md

[^15]: https://www.youtube.com/watch?v=0WH0yRxExA0\&vl=en

