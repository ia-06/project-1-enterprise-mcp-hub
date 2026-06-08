const METHODS = [
  {
    method: "system_customer360",
    desc:   "Fetch aggregated customer data from Salesforce, Jira, and Postgres simultaneously.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "salesforce_listAccounts",
    desc:   "List available Salesforce accounts to find valid Account IDs for the agent.",
    params: '{ "limit": 50 }',
  },
  {
    method: "salesforce_getAccount",
    desc:   "Fetch a single Salesforce Account by SF ID.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "jira_listTicketsByAccount",
    desc:   "List Jira tickets linked to a Salesforce Account ID.",
    params: '{ "accountSfId": "001..." }',
  },
  {
    method: "sales_listOrders",
    desc:   "List Postgres sales orders, optionally filtered by customer ID.",
    params: '{ "customerId": "uuid" }',
  },
  {
    method: "sales_getCustomerSummary",
    desc:   "Aggregate pipeline totals (closed-won, open, order count).",
    params: '{ "customerId": "uuid" }',
  },
];

const STEP_1 = `# Stdio mode (Claude/Cursor)
cd mcp-server
go run ./cmd/server -mode=stdio

# HTTP mode (VS Code Copilot)
go run ./cmd/server -mode=http`;

const STEP_2 = `# Claude Desktop / Cursor (stdio)
{
  "mcpServers": {
    "enterprise-hub": {
      "command": "go",
      "args": ["run", "./cmd/server", "-mode=stdio"]
    }
  }
}

# VS Code Copilot (http)
{
  "mcpServers": {
    "enterprise-hub": {
      "url": "http://localhost:8080/rpc",
      "type": "http"
    }
  }
}`;

const STEP_3 = `# Call tool via MCP Protocol
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "system_customer360",
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
          <div className="section-eyebrow" style={{ color: "rgba(255,255,255,0.35)" }}>
            Developer Reference
          </div>
          <h2 className="display-lg" style={{ color: "rgba(255,255,255,0.92)", marginBottom: "var(--space-md)" }}>
            Plug our MCP server into your agent workflow.
          </h2>
          <p className="body-lg" style={{ color: "rgba(255,255,255,0.45)" }}>
            The Go server natively supports both <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "13px", color: "#a5d6ff" }}>stdio</code> and <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "13px", color: "#a5d6ff" }}>http</code> transports. 
            Connect any MCP-compatible agent in three steps — then invoke the registered MCP tools below.
          </p>
        </div>

        {/* 3 steps */}
        <div className="mcp-step-grid">
          {[
            { num: "01", title: "Run the server", code: STEP_1 },
            { num: "02", title: "Configure your agent", code: STEP_2 },
            { num: "03", title: "Call a tool", code: STEP_3 },
          ].map((s) => (
            <div key={s.num} className="mcp-step-card">
              <div className="mcp-step-num">STEP {s.num}</div>
              <h3>{s.title}</h3>
              <div className="mcp-code">
                <pre>{s.code}</pre>
              </div>
            </div>
          ))}
        </div>

        {/* Methods table */}
        <div style={{ marginTop: "var(--space-3xl)" }}>
          <div
            style={{
              fontSize: "11px",
              fontFamily: "'JetBrains Mono', monospace",
              color: "rgba(255,255,255,0.3)",
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
                  <td style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "11px", color: "rgba(255,255,255,0.45)" }}>
                    {m.params}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Compatible agents note */}
        <div
          style={{
            marginTop: "var(--space-2xl)",
            padding: "var(--space-md) var(--space-lg)",
            background: "rgba(255,255,255,0.04)",
            borderRadius: "var(--r-md)",
            border: "1px solid rgba(255,255,255,0.08)",
          }}
        >
          <p style={{ fontSize: "13px", color: "rgba(255,255,255,0.45)", lineHeight: "20px" }}>
            <span style={{ color: "rgba(255,255,255,0.7)", fontWeight: 500 }}>Compatible agents:</span>{" "}
            GitHub Copilot, Claude Desktop, Cursor, Windsurf, or any custom agent that speaks the standard MCP protocol.
            The backend natively supports the initialize, tools/list, and tools/call lifecycles.
          </p>
        </div>
      </div>
    </section>
  );
}
