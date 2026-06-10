# Enterprise MCP Hub & Customer 360

A distributed architecture integrating Go, Next.js, and the Model Context Protocol (MCP) to unify enterprise data sources (Salesforce, Jira, PostgreSQL) into a highly available, concurrent middleware layer.

## System Topology

The backend simultaneously serves a React frontend and a JSON-RPC 2.0 MCP interface for autonomous AI agents (Claude, Cursor). 

```mermaid
flowchart TD
    Agent["AI Agent (Claude/Cursor)"]
    UI["Next.js App Router UI"]

    subgraph Hub["Enterprise MCP Hub (Go Middleware)"]
        direction TB
        RPC["JSON-RPC 2.0 Router"]
        MCP["MCP Interface Layer"]
        
        subgraph Concurrency["Concurrency Engine"]
            Batch["Batch Orchestrator"]
            Metrics["Metrics & Insights Engine"]
        end

        subgraph Adapters["Adapters"]
            AdapterSF["Salesforce API Adapter"]
            AdapterJira["Jira REST Adapter"]
            RepoSales["Postgres/Supabase Repository"]
        end
        
        RPC <--> MCP
        MCP <--> Batch
        Batch <--> Metrics
        Metrics <--> AdapterSF
        Metrics <--> AdapterJira
        Metrics <--> RepoSales
    end

    LiveSF[("Live Salesforce CRM")]
    LiveJira[("Live Jira Cloud")]
    Supabase[("Supabase Cache / DB")]

    UI <-->|HTTP / API| RPC
    Agent <-->|Stdio / HTTP JSON-RPC| RPC
    AdapterSF <-->|OAuth2 / REST| LiveSF
    AdapterJira <-->|HTTP Basic Auth| LiveJira
    RepoSales <-->|PostgREST| Supabase
    AdapterSF -.->|Background Async Upsert| Supabase
```

## Performance & N+1 Query Resolution

To prevent API exhaustion, a high-throughput **Batch Orchestrator** resolves O(N) queries into O(1) bulk operations.

```mermaid
sequenceDiagram
    participant Agent as AI Agent
    participant Hub as MCP Hub (Go)
    participant SF as Salesforce
    participant Jira as Jira Cloud
    participant PG as Supabase (Postgres)

    Agent->>Hub: Call system.customer360Batch(N Accounts)
    activate Hub
    
    par Salesforce Bulk Fetch
        Hub->>SF: ListAccountsByIDs (IN query)
    and Jira Bulk Fetch
        Hub->>Jira: ListTicketsByAccounts (JQL IN query)
    and Postgres Bulk Fetch
        Hub->>PG: GetCustomerSummaries (SQL IN query)
    end

    Hub->>Hub: Assemble N Customer 360 Maps & Deep Metrics
    Hub-->>Agent: Return Aggregated JSON Array
    deactivate Hub
```

## Graceful Resilience Degradation

Engineered to survive complete outages of critical infrastructure via an automated 30-minute background syncer and instant fallback caching.

```mermaid
stateDiagram-v2
    [*] --> QuerySalesforce
    QuerySalesforce --> LiveSuccess : HTTP 200 OK
    QuerySalesforce --> Timeout : HTTP 429 / 5xx / Network Error
    
    LiveSuccess --> CalculateMetrics
    CalculateMetrics --> AsyncCache : Fire and Forget Upsert
    CalculateMetrics --> ReturnData
    
    Timeout --> QuerySupabaseFallback
    QuerySupabaseFallback --> CacheHit : Data Exists
    QuerySupabaseFallback --> CacheMiss : Data Missing
    
    CacheHit --> FlagAsCached
    FlagAsCached --> ReturnData
    CacheMiss --> ReturnError
```

## AI Capabilities (MCP Tools)

The backend acts as an intelligent abstraction layer exposing 13 tools:
- **`system.customer360Batch` & `system.customer360`**: Unified concurrency aggregation.
- **`system.getAccountInsights`**: AI analysis of churn risk.
- **`system.healthCheck`**: Backend resilience probing.
- **`salesforce.*`**: Search, List, Get Accounts.
- **`jira.*`**: List tickets, fetch trends, and escalate (`jira.escalateTicket`).
- **`sales.*`**: List orders, get summaries, adjust limits (`system.adjustApiRateLimit`).

**Connect via Stdio (Claude Desktop / Cursor):**
```json
{
  "mcpServers": {
    "enterprise-hub": {
      "command": "go",
      "args": ["run", "./cmd/server", "-mode=stdio"]
    }
  }
}
```

## Quickstart

1. **Clone**: `git clone https://github.com/ia-06/project-1-enterprise-mcp-hub.git`
2. **Config**: `cp .env.example .env` *(Set `SF_USE_MOCK=true` and `JIRA_USE_MOCK=true` to bypass live APIs)*
3. **Database**: `cd infra && docker compose up --build -d`
4. **Frontend**: `npm install && npm run dev`
5. **Verify**: Visit `http://localhost:3000`.