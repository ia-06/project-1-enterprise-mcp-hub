const CONNECTORS = [
  {
    name: "Jira Cloud",
    tag: "Atlassian REST API v3",
    color: "#0052cc",
    steps: [
      {
        title: "Create Atlassian account + API token",
        body: (
          <>
            Sign up at{" "}
            <a href="https://www.atlassian.com/try/cloud/signup" target="_blank" rel="noopener noreferrer">
              atlassian.com
            </a>
            , create a Jira Software project, then generate a personal API token at{" "}
            <a href="https://id.atlassian.com/manage-profile/security/api-tokens" target="_blank" rel="noopener noreferrer">
              id.atlassian.com → Security → API tokens
            </a>
            .
          </>
        ),
      },
      {
        title: "Link Jira issues to Salesforce accounts",
        body: "Add a custom text field named 'Salesforce Account ID' to your Jira project. Set the field value on each issue to the corresponding Salesforce Account.Id (15–18 character ID). The adapter auto-resolves the JQL field key — no manual cf[] IDs needed.",
      },
      {
        title: "Set environment variables",
        body: null,
        code: `JIRA_BASE_URL="https://<your-domain>.atlassian.net"
JIRA_EMAIL="you@company.com"
JIRA_API_TOKEN="ATATT3xFfGF0..."
JIRA_PROJECT_KEY="P1EMH"
JIRA_SF_ACCOUNT_FIELD="Salesforce Account ID"
JIRA_USE_MOCK="false"`,
      },
    ],
  },
  {
    name: "Salesforce CRM",
    tag: "OAuth2 ROPC",
    color: "#00a1e0",
    steps: [
      {
        title: "Create Developer Edition org",
        body: (
          <>
            Sign up at{" "}
            <a href="https://developer.salesforce.com/signup" target="_blank" rel="noopener noreferrer">
              developer.salesforce.com
            </a>
            . Log in, reset your Security Token via{" "}
            <strong>Avatar → Settings → Reset My Security Token</strong>. Check email for the token.
          </>
        ),
      },
      {
        title: "Create a Connected App",
        body: "In Salesforce Setup → App Manager → New Connected App. Enable OAuth, set callback to localhost, add Full Access scope. After saving (~10 min), open Manage Consumer Details and copy the Consumer Key + Secret.",
      },
      {
        title: "Set environment variables",
        body: null,
        code: `SF_BASE_URL="https://<org>.my.salesforce.com"
SF_LOGIN_URL="https://login.salesforce.com"
SF_CLIENT_ID="3MVG9..."
SF_CLIENT_SECRET="ABC123..."
SF_USERNAME="you@sandbox.dev"
SF_PASSWORD="MyPassSecurityToken"
SF_API_VERSION="v66.0"
SF_USE_MOCK="false"`,
      },
    ],
  },
  {
    name: "Supabase",
    tag: "PostgREST + New API Keys",
    color: "#3ecf8e",
    steps: [
      {
        title: "Create project + run schema",
        body: (
          <>
            Create a project at{" "}
            <a href="https://supabase.com" target="_blank" rel="noopener noreferrer">
              supabase.com
            </a>
            . In the SQL Editor, run{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>
              db/migrations/001_create_sales_tables.sql
            </code>{" "}
            then{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>
              db/seed/sales_seed.sql
            </code>
            .
          </>
        ),
      },
      {
        title: "Copy new-format API keys",
        body: "Go to Settings → API Keys. Copy the Publishable key (sb_publishable_…) for read-only PostgREST queries and the Secret key (sb_secret_…) for server-side admin ops. Do NOT use the legacy anon/service_role keys — they are deprecated.",
      },
      {
        title: "Set environment variables",
        body: null,
        code: `SUPABASE_URL="https://<ref>.supabase.co"
SUPABASE_PUBLISHABLE_KEY="sb_publishable_..."
SUPABASE_SECRET_KEY="sb_secret_..."
PG_DSN="postgres://postgres:<pass>@db.<ref>.supabase.co:5432/postgres"
SUPABASE_ENABLED="true"`,
      },
    ],
  },
];

export default function ConnectSection() {
  return (
    <section className="section" id="connect" style={{ background: "var(--canvas)" }}>
      <div className="container">
        {/* Header */}
        <div style={{ maxWidth: "600px", marginBottom: "var(--space-3xl)" }}>
          <div className="section-eyebrow">Setup guide</div>
          <h2 className="display-lg" style={{ color: "var(--ink)", marginBottom: "var(--space-md)" }}>
            Connect your enterprise accounts.
          </h2>
          <p className="body-lg" style={{ color: "var(--body)" }}>
            All credentials live in your server-side{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "14px", color: "var(--ink)", background: "var(--canvas-soft-2)", padding: "2px 6px", borderRadius: "4px" }}>
              .env
            </code>{" "}
            file — never in the browser. Follow the steps below for each channel.
          </p>
        </div>

        {/* 3-col cards */}
        <div className="connect-grid">
          {CONNECTORS.map((c) => (
            <div key={c.name} className="connect-card">
              {/* Card header */}
              <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
                <div
                  style={{
                    width: "8px",
                    height: "8px",
                    borderRadius: "50%",
                    background: c.color,
                    flexShrink: 0,
                  }}
                />
                <div>
                  <div style={{ fontSize: "15px", fontWeight: 600, color: "var(--ink)", letterSpacing: "-0.3px" }}>
                    {c.name}
                  </div>
                  <div className="caption-mono" style={{ color: "var(--mute)", marginTop: "2px" }}>
                    {c.tag}
                  </div>
                </div>
              </div>

              {/* Steps */}
              <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-md)" }}>
                {c.steps.map((step, i) => (
                  <div key={i} className="connect-step">
                    <div className="connect-step-num">{i + 1}</div>
                    <div className="connect-step-body">
                      <div className="connect-step-title">{step.title}</div>
                      {step.body && (
                        <div className="connect-step-text">{step.body}</div>
                      )}
                      {step.code && (
                        <div className="code-block" style={{ marginTop: "var(--space-sm)" }}>
                          <pre>{step.code}</pre>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Security note */}
        <div
          style={{
            marginTop: "var(--space-2xl)",
            background: "var(--canvas-soft)",
            border: "1px solid var(--hairline)",
            borderRadius: "var(--r-md)",
            padding: "var(--space-md) var(--space-lg)",
            display: "flex",
            alignItems: "center",
            gap: "var(--space-sm)",
          }}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: "var(--mute)", flexShrink: 0 }}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
          </svg>
          <p className="body-sm" style={{ color: "var(--body)" }}>
            <strong style={{ color: "var(--ink)" }}>Security note:</strong>{" "}
            All API keys and OAuth credentials belong in your backend{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>.env</code> file or a
            secrets manager (AWS Secrets Manager, HashiCorp Vault). Never expose them to the browser or commit
            them to version control. The <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>.gitignore</code> already excludes{" "}
            <code style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>.env</code>.
          </p>
        </div>
      </div>
    </section>
  );
}
