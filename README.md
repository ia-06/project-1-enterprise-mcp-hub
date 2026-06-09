# Enterprise MCP Hub & Customer 360

A distributed systems architecture integrating Go, Next.js, and the Model Context Protocol (MCP).

## Overview

The Enterprise MCP Hub is a backend aggregation layer designed to unify disparate enterprise data sources (Salesforce, Jira, PostgreSQL) into a highly available service. It provides a concurrent data-fetching pipeline and implements the Model Context Protocol (MCP) to allow direct orchestration by AI agents.

## Architecture

The system is designed with a decoupled architecture focusing on concurrency, resilience, and agent interoperability.

### 1. Concurrency Engine (Go Backend)
The backend operates as a middle-tier orchestrator. Requests to the `system_customer360` endpoint are processed using `sync.WaitGroup` to execute concurrent API calls against Salesforce (REST API), Jira (Atlassian API), and PostgreSQL. Data is aggregated, scored, and returned in a unified JSON structure.

### 2. Resilient Fallback Strategy
To address external API rate limits and downtime, the Salesforce adapter implements a layered cache architecture. If the primary Salesforce API is unreachable, the system degrades gracefully, utilizing a Supabase Edge Cache to fulfill requests without interrupting the client connection.

### 3. Model Context Protocol (MCP) Integration
The backend integrates a protocol adapter within its JSON-RPC router. Standard MCP clients can autonomously invoke `initialize`, `tools/list`, and `tools/call`. This allows autonomous AI agents, such as GitHub Copilot, to orchestrate backend workflows and query enterprise data natively.

### 4. Next.js Client & Auto-Seeder
The frontend is a Next.js application that visualizes the aggregated data streams via a responsive UI framework. It features an integrated **Data Seeder Engine** (`/api/seed`) that safely wipes and regenerates highly-complex synthetic data (including dynamic MRR logic and comprehensive Jira workflow state transitions) to ensure realistic datasets for AI orchestration testing.

## Advanced Data Modeling

To provide realistic data scenarios for the connected MCP Agent, the Hub automatically generates rich datasets for every account:
- **Jira Workflow Statuses:** The seeder automatically transitions AI-generated `Task` issues across a real, custom software lifecycle: `Todo` → `In Progress` → `Build Broken` | `Building` → `Done` → `Reopened`.
- **Health Score Algorithm:** A heavily optimized 100-point algorithm calculates live health by integrating:
  - Total Monthly Recurring Revenue (MRR)
  - Sales Pipeline Velocity (`CLOSED_WON` vs `CLOSED_LOST` ratios)
  - Architectural Churn (Penalizing tickets stuck in `Build Broken` or `Reopened` states)

## AI Agent Integration (MCP)

The backend functions as a Universal MCP Server, supporting both `stdio` and `http` transports.

### Available MCP Tools

Agents connected to the hub will natively discover the following tools:
- `system.customer360`: Concurrently aggregates SF, Jira, and PG data for a unified view.
- `system.accountMetrics`: **[NEW]** Provides pre-calculated mathematical probabilities, ticket-to-order ratios, churn risk percentages, and MRR. Eliminates basic arithmetic load off the LLM context window.
- `salesforce.getAccount`: Resolves direct metadata from Salesforce.
- `jira.listTicketsByAccount`: Pulls active Jira backlog items scoped to specific SF domains.
- `sales.listOrders`: Resolves Postgres pipeline tables.

### Option A: Standard Stdio Transport (Claude Desktop, Cursor, Windsurf)
Configure your agent to execute the Go binary as a child process using the `-mode=stdio` flag.
Example configuration:
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

### Option B: HTTP Transport (VS Code Copilot)
If the server is running in HTTP mode (default), configure the agent to connect via the RPC endpoint:
```json
{
  "mcpServers": {
    "enterprise-hub": {
      "url": "http://localhost:8080/rpc",
      "type": "http"
    }
  }
}
```

## Simulated Complex Prompts

By combining the unified view with the `system.accountMetrics` tool, the MCP Agent is fully capable of answering high-level strategic questions. Try asking your agent these complex prompts:

1. *"Analyze the relationship between Accounts with `Build Broken` or `Reopened` Jira tickets and their respective MRR churn probability. Which account has the highest statistical risk of abandoning their pipeline based on the ticket-to-order ratio provided by your metrics tool?"*

2. *"Look at the top 3 highest-MRR accounts. Evaluate their recent sales velocity against their Jira workload friction. Should we allocate more engineering support to any of these VIP clients to prevent revenue churn?"*

---

## Quickstart Setup

The following guide outlines the standard procedure to deploy the infrastructure locally.

### Prerequisites
- Docker and Docker Compose
- Node.js (v18+)
- Go (1.22+)

### Step 1: Clone the Repository
```bash
git clone https://github.com/ia-06/project-1-enterprise-mcp-hub.git
cd project-1-enterprise-mcp-hub
```

### Step 2: Configure Environment Variables
Copy the example environment configuration and populate the credentials for Salesforce, Jira, and Supabase.
*(Note: To bypass live external API dependencies, configure `SF_USE_MOCK=true` and `JIRA_USE_MOCK=true` in the `.env` file).*
```bash
cp .env.example .env
```

### Step 3: Initialize Infrastructure
Use Docker Compose to provision the Go MCP server and the PostgreSQL database. The database will automatically execute initialization scripts to seed the required tables.
```bash
cd infra
docker compose up --build -d
```

### Step 4: Launch Frontend
Return to the project root to install dependencies and start the Next.js development server.
```bash
cd ..
npm install
npm run dev
```

### Step 5: Verification
Navigate to `http://localhost:3000` to verify the application is successfully streaming data from the backend. If your dataset is empty, the Dashboard will automatically prompt you to run the Data Seeder.