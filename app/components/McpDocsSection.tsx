const METHODS = [
  {
    method: "system.customer360",
    desc:   "Fetch aggregated customer data from Salesforce, Jira, and Postgres simultaneously.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "system.customer360Batch",
    desc:   "Get a complete Customer 360 view for MULTIPLE accounts simultaneously. Highly optimized bulk API call to prevent N+1 query exhaustion.",
    params: '{ "accountIds": "001ACME001,001ACME002" }',
  },
  {
    method: "system.getAccountInsights",
    desc:   "Analyzes ticket-to-order ratio, broken ticket percentage, and MRR, and returns a plain English string detailing why the account is failing and recommended actions.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "system.adjustApiRateLimit",
    desc:   "Dynamically alters a client's API limits in the Postgres database via Supabase PostgREST.",
    params: '{ "accountId": "001...", "newLimit": 5000 }',
  },
  {
    method: "system.healthCheck",
    desc:   "Probe all backend adapter statuses (Salesforce, Jira, Postgres) to verify system resilience matrix. Does NOT reflect fintech client accounts health.",
    params: '{}',
  },
  {
    method: "salesforce.searchAccounts",
    desc:   "Search Salesforce Accounts by name. Returns up to 50 matching records.",
    params: '{ "query": "Acme" }',
  },
  {
    method: "salesforce.listAccounts",
    desc:   "List available Salesforce accounts to find valid Account IDs for the agent.",
    params: '{ "limit": 50 }',
  },
  {
    method: "salesforce.getAccount",
    desc:   "Fetch a single Salesforce Account by SF ID.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "jira.listTicketsByAccount",
    desc:   "List Jira tickets linked to a Salesforce Account ID.",
    params: '{ "accountSfId": "001..." }',
  },
  {
    method: "sales.listOrders",
    desc:   "List Postgres sales orders, optionally filtered by customer ID.",
    params: '{ "customerId": "uuid" }',
  },
  {
    method: "sales.getCustomerSummary",
    desc:   "Aggregate pipeline totals (closed-won, open, order count).",
    params: '{ "customerId": "uuid" }',
  },
  {
    method: "jira.getAccountTicketTrends",
    desc:   "Analyzes open tickets and summarizes the primary issue categories for a given account.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "jira.escalateTicket",
    desc:   "Updates a Jira issue's priority via the Atlassian REST API and adds an escalation comment.",
    params: '{ "ticketKey": "ENG-123", "newPriority": "Highest" }',
  },
];

const STEP_1 = `# Compile the Go binary (Crucial for stdio)
npm run build:mcp

# The binary is now ready at:
# ./mcp-server/mcp-hub.exe (Windows)
# ./mcp-server/mcp-hub (Mac/Linux)`;

const STEP_2 = `# Cursor / Claude / Windsurf (stdio)
{
  "mcpServers": {
    "enterprise-hub": {
      "command": "/absolute/path/to/mcp-server/mcp-hub",
      "args": ["-mode=stdio"],
      "cwd": "/absolute/path/to/project-1-enterprise-mcp-hub"
    }
  }
}

# DO NOT use "go run" in the command! 
# Go compiler stdout logs will fatally corrupt JSON-RPC.`;

const STEP_3 = `# Call tool via MCP Protocol
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "system.customer360",
    "arguments": {
      "accountId": "001abc..."
    }
  },
  "id": "1"
}

# Response:
{
  "jsonrpc": "2.0",
  "result": {
    "content": [{ "type": "text", "text": "{...}" }]
  },
  "id": "1"
}`;

export default function McpDocsSection() {
  return (
    <section className="section section-dark" id="mcp-docs">
      <div className="container">
        {/* Header */}
        <div style={{ maxWidth: "640px", marginBottom: "var(--space-3xl)" }}>
          <div className="section-eyebrow">
            Developer Reference
          </div>
          <h2 className="display-lg" style={{ color: "var(--ink)", marginBottom: "var(--space-md)" }}>
            Plug our MCP server into your agent workflow.
          </h2>
          <p className="body-lg" style={{ color: "var(--body)" }}>
            The Go server natively supports both <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "13px", color: "var(--link)" }}>stdio</code> and <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "13px", color: "var(--link)" }}>http</code> transports. 
            Connect any MCP-compatible agent in three steps — then invoke the registered MCP tools below.
          </p>
        </div>

        {/* 3 steps */}
        <div className="mcp-step-grid">
          <div className="mcp-step-card">
            <div className="mcp-step-num">STEP 01</div>
            <h3>Compile the Binary</h3>
            <p>Always compile the server before connecting your agent. Relying on <code>go run</code> can pollute stdout and fatally corrupt the JSON-RPC handshake.</p>
            <div className="mcp-code">
              <pre>{STEP_1}</pre>
            </div>
          </div>

          <div className="mcp-step-card">
            <div className="mcp-step-num">STEP 02</div>
            <h3>Configure your Agent</h3>
            <p>Provide the absolute path to the compiled binary in your IDE&apos;s MCP configuration. The agent will auto-discover tools upon connection.</p>
            <div className="mcp-code">
              <pre>{STEP_2}</pre>
            </div>
          </div>

          <div className="mcp-step-card">
            <div className="mcp-step-num">STEP 03</div>
            <h3>Invoke Native Tools</h3>
            <p>The agent can now read and write directly to Salesforce, Jira, and Postgres using the provided RPC methods.</p>
            <div className="mcp-code">
              <pre>{STEP_3}</pre>
            </div>
          </div>
        </div>

        {/* Methods table */}
        <div style={{ marginTop: "var(--space-3xl)" }}>
          <div
            style={{
              fontSize: "11px",
              fontFamily: "'JetBrains Mono', monospace",
              color: "var(--mute)",
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              marginBottom: "var(--space-md)",
            }}
          >
            Available Methods
          </div>
          <table className="methods-table">
            <thead>
              <tr>
                <th>Method</th>
                <th>Description</th>
                <th>Example Params</th>
              </tr>
            </thead>
            <tbody>
              {METHODS.map((m) => (
                <tr key={m.method}>
                  <td>{m.method}</td>
                  <td>{m.desc}</td>
                  <td style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "11px", color: "var(--mute)" }}>
                    {m.params}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Compatible agents note */}
        <div style={{ 
          marginTop: "var(--space-3xl)", 
          background: "var(--canvas-soft-2)", 
          borderRadius: "var(--r-md)", 
          border: "1px solid var(--hairline)", 
          padding: "var(--space-lg)" 
        }}>
          <p style={{ fontSize: "13px", color: "var(--body)", lineHeight: "20px" }}>
            <span style={{ color: "var(--ink)", fontWeight: 500 }}>Compatible agents:</span>{" "}
            GitHub Copilot, Claude Desktop, Cursor, Windsurf, or any custom agent that speaks the standard MCP protocol. 
            The backend natively supports the <code className="code-text" style={{fontSize: "11px", color: "var(--link)"}}>initialize</code>, <code className="code-text" style={{fontSize: "11px", color: "var(--link)"}}>tools/list</code>, and <code className="code-text" style={{fontSize: "11px", color: "var(--link)"}}>tools/call</code> lifecycles.
          </p>
        </div>
      </div>
    </section>
  );
}
