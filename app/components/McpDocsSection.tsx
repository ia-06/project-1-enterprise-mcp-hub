const METHODS = [
  {
    method: "salesforce.listAccounts",
    desc:   "List live Salesforce Accounts (SOQL). Populates the dashboard selector.",
    params: '{ "limit": 50 }',
  },
  {
    method: "salesforce.getAccount",
    desc:   "Fetch a single Account by SF ID. Auto-falls back to Supabase cache.",
    params: '{ "accountId": "001..." }',
  },
  {
    method: "jira.listTicketsByAccount",
    desc:   "JQL tickets linked to an SF Account ID via the custom field.",
    params: '{ "accountSfId": "001..." }',
  },
  {
    method: "sales.listOrders",
    desc:   "Postgres sales orders, optionally filtered by internal customer UUID.",
    params: '{ "customerId": "uuid" }',
  },
  {
    method: "sales.getCustomerSummary",
    desc:   "Aggregate pipeline totals (closed-won, open, order count).",
    params: '{ "customerId": "uuid" }',
  },
  {
    method: "system.healthCheck",
    desc:   "Probes all three channels and returns status for each.",
    params: "{}",
  },
];

const STEP_1 = `# 1. Start the MCP server
cd mcp-server
go run ./cmd/server
# > [fiber] listening on :8080`;

const STEP_2 = `// 2. Configure your agent's MCP endpoint
{
  "mcpServer": {
    "url": "http://localhost:8080/rpc",
    "transport": "jsonrpc"
  }
}`;

const STEP_3 = `// 3. Call any tool via JSON-RPC 2.0
{
  "jsonrpc": "2.0",
  "method": "salesforce.getAccount",
  "params": { "accountId": "001abc..." },
  "id": "1"
}
// Response:
{
  "jsonrpc": "2.0",
  "result": {
    "account": {
      "id": "001abc...",
      "name": "ACME Corp",
      "industry": "Manufacturing",
      "source": "live"
    }
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
            The Go Fiber server exposes JSON-RPC 2.0 at{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "13px", color: "#a5d6ff" }}>
              POST /rpc
            </code>{" "}
            on port 8080. Connect any MCP-compatible agent in three steps — then call any of the
            six registered tools below.
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
            Claude Desktop, Cursor, Windsurf, GitHub Copilot Extensions, LangChain JSON-RPC tool,
            or any custom agent that speaks the MCP JSON-RPC 2.0 protocol.
            Batch requests (JSON array) are fully supported.
          </p>
        </div>
      </div>
    </section>
  );
}
