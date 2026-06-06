"use client";

import React from "react";
import type { SystemHealth } from "@/app/lib/types";

type Props = {
  health: SystemHealth | null;
};

/**
 * SystemHealthBanner
 *
 * Renders a sticky banner at the top of the dashboard reflecting
 * the aggregate health of all backing services.
 *
 * Resilience states:
 *   - All up           → green dot, "All systems operational"
 *   - Any degraded     → yellow dot + warning message (Scenario A cache state)
 *   - Any down         → red dot + error message
 *   - null / unknown   → grey dot, "System health unknown"
 */
export default function SystemHealthBanner({ health }: Props) {
  // -------------------------------------------------------------------------
  // Derive aggregate status
  // -------------------------------------------------------------------------
  let bannerVariant: "healthy" | "degraded" | "critical" | "unknown" = "unknown";
  let dotVariant:    "green" | "yellow" | "red" | "gray" = "gray";
  let headline = "Checking system health…";
  let subline: string | null = null;

  if (health) {
    const services = [health.mcpServer, health.sales, health.jira, health.salesforce];
    const hasDown     = services.some((s) => s.status === "down");
    const hasDegraded = services.some((s) => s.status === "degraded");
    const sfCached    = health.salesforce?.cached;

    if (hasDown) {
      bannerVariant = "critical";
      dotVariant    = "red";
      headline      = "One or more services are down";
      subline       = "Data may be incomplete. Retry or contact your platform team.";
    } else if (hasDegraded || sfCached) {
      bannerVariant = "degraded";
      dotVariant    = "yellow";
      headline      = "Salesforce degraded — serving cached data";
      subline       = sfCached
        ? "Salesforce CRM is unreachable. Account data is sourced from the Supabase snapshot cache."
        : "Some services are running in degraded mode.";
    } else {
      bannerVariant = "healthy";
      dotVariant    = "green";
      headline      = "All systems operational";
    }
  }

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------
  return (
    <div
      className={`mcp-health-banner mcp-health-banner--${bannerVariant}`}
      role="status"
      aria-live="polite"
      aria-label="System health status"
    >
      <span
        className={`mcp-health-banner-dot mcp-health-banner-dot--${dotVariant}`}
        aria-hidden="true"
      />

      <span className="mcp-health-banner-message">
        <strong>{headline}</strong>
        {subline && (
          <>
            {" · "}
            <span>{subline}</span>
          </>
        )}
      </span>

      {/* Per-service chips — only render when health data is available */}
      {health && (
        <div className="mcp-health-banner-services" aria-label="Individual service statuses">
          <ServiceChip label="MCP"        status={health.mcpServer?.status ?? "unknown"} />
          <ServiceChip label="Postgres"   status={health.sales?.status ?? "unknown"} />
          <ServiceChip label="Jira"       status={health.jira?.status ?? "unknown"} />
          <ServiceChip label="Salesforce" status={health.salesforce?.status ?? "unknown"} cached={health.salesforce?.cached} />
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sub-component: individual service chip
// ---------------------------------------------------------------------------
function ServiceChip({
  label,
  status,
  cached,
}: {
  label: string;
  status: string;
  cached?: boolean;
}) {
  const chipVariant =
    status === "up"       ? "up"       :
    status === "degraded" ? "degraded" :
    status === "down"     ? "down"     : "unknown";

  const displayStatus = cached ? "cached" : status;

  return (
    <span
      className={`mcp-health-service-chip mcp-health-service-chip--${chipVariant}`}
      title={`${label}: ${displayStatus}`}
      aria-label={`${label} status: ${displayStatus}`}
    >
      {label}: {displayStatus}
    </span>
  );
}
