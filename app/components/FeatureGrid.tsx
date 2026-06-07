const FEATURES = [
  {
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
      </svg>
    ),
    name: "Jira Cloud",
    tag: "REST API v3",
    description:
      "Live engineering tickets fetched via Jira Cloud REST API v3 with HTTP Basic Auth. Tickets are dynamically filtered by Salesforce Account ID using an auto-resolved JQL custom field — no hardcoded IDs.",
    detail: "jira.listTicketsByAccount",
  },
  {
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
      </svg>
    ),
    name: "Salesforce CRM",
    tag: "OAuth2 ROPC",
    description:
      "Real-time Account data via OAuth2 Resource Owner Password Credentials flow. Automatic Supabase cache fallback activates when Salesforce exceeds the request timeout — Scenario A resilience built in.",
    detail: "salesforce.getAccount",
  },
  {
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
      </svg>
    ),
    name: "Postgres / Supabase",
    tag: "PostgREST",
    description:
      "Direct Supabase PostgREST queries for customers, sales orders, and pipeline aggregates. Uses the new publishable key system (sb_publishable_…). Doubles as the Salesforce fallback cache layer.",
    detail: "sales.listOrders",
  },
];

export default function FeatureGrid() {
  return (
    <section className="section" id="features" style={{ background: "var(--canvas-soft)" }}>
      <div className="container">
        {/* Header */}
        <div className="section-header text-center" style={{ maxWidth: "600px", margin: "0 auto var(--space-3xl)" }}>
          <div className="section-eyebrow">What's inside</div>
          <h2 className="display-lg" style={{ color: "var(--ink)", marginBottom: "var(--space-md)" }}>
            One middleware. Three live channels.
          </h2>
          <p className="body-lg" style={{ color: "var(--body)" }}>
            Every method call hits a real API. No fixtures, no stubs — the same adapter logic that an LLM agent uses is
            what drives the dashboard.
          </p>
        </div>

        {/* Grid */}
        <div className="feature-grid">
          {FEATURES.map((f) => (
            <div key={f.name} className="feature-card">
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                <div className="feature-icon" style={{ color: "var(--body)" }}>
                  {f.icon}
                </div>
                <span className="badge badge-mono">{f.tag}</span>
              </div>
              <div>
                <h3>{f.name}</h3>
                <p style={{ marginTop: "var(--space-xs)" }}>{f.description}</p>
              </div>
              <div style={{ paddingTop: "var(--space-sm)", borderTop: "1px solid var(--hairline)" }}>
                <span className="caption-mono" style={{ color: "var(--mute)" }}>
                  {f.detail}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
