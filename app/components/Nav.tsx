"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

const LINKS = [
  { label: "Features",  href: "/#features" },
  { label: "Connect",   href: "/#connect" },
  { label: "MCP Docs",  href: "/#mcp-docs" },
  { label: "Dashboard", href: "/dashboard" },
];

export default function Nav() {
  const path = usePathname();

  return (
    <nav className="nav">
      <div className="nav-inner">
        {/* Brand */}
        <Link href="/" className="nav-brand" style={{ textDecoration: "none" }}>
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <rect width="20" height="20" rx="5" fill="#171717" />
            <path d="M5 14L10 6L15 14H5Z" fill="white" />
          </svg>
          Enterprise MCP Hub
        </Link>

        {/* Center links */}
        <div className="nav-links">
          {LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={`nav-link${path === link.href ? " active" : ""}`}
            >
              {link.label}
            </Link>
          ))}
        </div>

        {/* CTA cluster */}
        <div className="nav-actions">
          <a
            href="https://github.com"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-nav btn-nav-secondary"
          >
            GitHub
          </a>
          <Link href="/dashboard" className="btn btn-nav btn-nav-primary">
            Open Dashboard
          </Link>
        </div>
      </div>
    </nav>
  );
}
