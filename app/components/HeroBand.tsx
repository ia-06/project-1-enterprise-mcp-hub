import Link from "next/link";

export default function HeroBand() {
  return (
    <section className="hero" id="hero">
      {/* Atmospheric mesh gradient — hero scale only */}
      <div className="hero-gradient" aria-hidden="true" />

      {/* Badge */}
      <div style={{ display: "flex", justifyContent: "center", marginBottom: "var(--space-xl)" }}>
        <div className="hero-badge">
          <span className="hero-badge-dot" aria-hidden="true" />
          Enterprise MCP Hub · v1.0 · Production-Ready
        </div>
      </div>

      {/* Headline */}
      <h1 className="hero-title">
        Connect your enterprise data stack. In one middleware.
      </h1>

      {/* Lead */}
      <p className="hero-subtitle">
        A production-grade MCP server that wires Jira Cloud, Salesforce CRM, and
        Postgres into any AI agent workflow via JSON-RPC&nbsp;2.0. No mocks. Live APIs.
      </p>

      {/* CTA row */}
      <div className="hero-cta-row">
        <Link href="/dashboard" className="btn btn-primary" style={{ height: "44px", padding: "0 24px", fontSize: "15px" }}>
          Open Dashboard
        </Link>
        <Link href="/#connect" className="btn btn-secondary" style={{ height: "44px", padding: "0 24px", fontSize: "15px" }}>
          Connect Your Stack
        </Link>
        <a
          href="/#mcp-docs"
          className="btn btn-secondary"
          style={{ height: "44px", padding: "0 24px", fontSize: "15px" }}
        >
          MCP Docs →
        </a>
      </div>

      {/* Stats strip */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: "var(--space-3xl)",
          marginTop: "var(--space-5xl)",
          flexWrap: "wrap",
        }}
      >
        {[
          { value: "3",        label: "Live Data Channels" },
          { value: "5",        label: "JSON-RPC Methods" },
          { value: "< 50ms",   label: "Median Response" },
          { value: "99.9%",    label: "Uptime Target" },
        ].map((s) => (
          <div key={s.label} style={{ textAlign: "center" }}>
            <div style={{ fontSize: "28px", fontWeight: 600, letterSpacing: "-1px", color: "var(--ink)" }}>
              {s.value}
            </div>
            <div style={{ fontSize: "12px", color: "var(--mute)", marginTop: "4px", fontFamily: "'JetBrains Mono', monospace" }}>
              {s.label}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
