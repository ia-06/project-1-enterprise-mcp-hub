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

### 4. Next.js Client
The frontend is a Next.js application that visualizes the aggregated data streams via a responsive UI framework.

## Quickstart Setup

The following guide outlines the standard procedure to deploy the infrastructure locally.

### Prerequisites
- Docker and Docker Compose
- Node.js (v18+)
- Go (1.22+)

### Step 1: Clone the Repository
```bash
git clone https://github.com/your-org/project-1-enterprise-mcp-hub.git
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
Navigate to `http://localhost:3000` to verify the application is successfully streaming data from the backend.

## AI Agent Integration (MCP)

The backend functions as a Universal MCP Server, supporting both `stdio` and `http` transports.

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

Upon connection, the agent will have autonomous access to the full suite of enterprise tools (e.g., `system_customer360`, `salesforce_getAccount`, `jira_listTicketsByAccount`, `sales_listOrders`).

---