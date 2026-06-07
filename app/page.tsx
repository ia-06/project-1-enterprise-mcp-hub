import HeroBand from "./components/HeroBand";
import FeatureGrid from "./components/FeatureGrid";
import ConnectSection from "./components/ConnectSection";
import McpDocsSection from "./components/McpDocsSection";

// Logo strip data — monochrome SVG text representations
const LOGOS = [
  { name: "Atlassian",   abbr: "ATLASSIAN" },
  { name: "Salesforce",  abbr: "SALESFORCE" },
  { name: "Supabase",    abbr: "SUPABASE" },
  { name: "PostgreSQL",  abbr: "POSTGRESQL" },
  { name: "Go",          abbr: "GO" },
  { name: "Next.js",     abbr: "NEXT.JS" },
];

export default function HomePage() {
  return (
    <main>
      {/* 1 — Hero */}
      <HeroBand />

      {/* 2 — Logo strip */}
      <div className="logo-strip">
        <p className="logo-strip-label">Integrates with your existing enterprise stack</p>
        <div className="logo-strip-inner">
          {LOGOS.map((l) => (
            <span
              key={l.name}
              style={{
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: "11px",
                fontWeight: 400,
                color: "var(--hairline-strong)",
                letterSpacing: "0.12em",
                userSelect: "none",
              }}
            >
              {l.abbr}
            </span>
          ))}
        </div>
      </div>

      {/* 3 — Feature grid */}
      <FeatureGrid />

      {/* 4 — Connect / Setup guide */}
      <ConnectSection />

      {/* 5 — MCP Docs (dark band) */}
      <McpDocsSection />
    </main>
  );
}
