import Link from "next/link";

const COLS = [
  {
    title: "Product",
    links: [
      { label: "Features",  href: "/#features" },
      { label: "Connect",   href: "/#connect" },
      { label: "MCP Docs",  href: "/#mcp-docs" },
      { label: "Dashboard", href: "/dashboard" },
    ],
  },
  {
    title: "Integrations",
    links: [
      { label: "Jira Cloud",   href: "https://developer.atlassian.com", ext: true },
      { label: "Salesforce",   href: "https://developer.salesforce.com", ext: true },
      { label: "Supabase",     href: "https://supabase.com", ext: true },
      { label: "PostgreSQL",   href: "https://www.postgresql.org", ext: true },
    ],
  },
  {
    title: "Developer",
    links: [
      { label: "JSON-RPC Schema",  href: "/#mcp-docs" },
      { label: "Go MCP Server",    href: "/#mcp-docs" },
      { label: "REST API v3",      href: "/#connect" },
      { label: "OAuth2 Guide",     href: "/#connect" },
    ],
  },
];

export default function Footer() {
  return (
    <footer className="footer">
      <div className="footer-inner">
        {/* Brand col */}
        <div>
          <div className="footer-brand-name">Enterprise MCP Hub</div>
          <p className="footer-tagline">
            A production-grade Model Context Protocol server connecting
            Jira, Salesforce, and Postgres into any AI agent workflow.
          </p>
        </div>

        {COLS.map((col) => (
          <div key={col.title}>
            <div className="footer-col-title">{col.title}</div>
            <ul className="footer-links">
              {col.links.map((link) => (
                <li key={link.label}>
                  {"ext" in link && link.ext ? (
                    <a href={link.href} target="_blank" rel="noopener noreferrer">
                      {link.label}
                    </a>
                  ) : (
                    <Link href={link.href}>{link.label}</Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      <div className="footer-bottom">
        <span className="footer-bottom-text">
          © {new Date().getFullYear()} Enterprise MCP Hub. MIT License.
        </span>
        <span className="footer-bottom-text" style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "11px" }}>
          JSON-RPC 2.0 · Go Fiber · Next.js 14
        </span>
      </div>
    </footer>
  );
}
