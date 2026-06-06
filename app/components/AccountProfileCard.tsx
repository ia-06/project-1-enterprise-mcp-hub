"use client";

import React from "react";
import type { Account } from "@/app/lib/types";

type Props = {
  account: Account | null;
};

/**
 * AccountProfileCard
 *
 * Displays the Salesforce Account profile for the selected customer.
 *
 * Resilience behaviors:
 *   - account === null  → placeholder / skeleton state
 *   - account.source === "cache" → renders "Cached snapshot" badge (Scenario A)
 *   - account.source === "live"  → renders "Live" badge
 */
export default function AccountProfileCard({ account }: Props) {
  if (!account) {
    return (
      <article className="mcp-card mcp-account-profile-card" aria-label="Account Profile">
        <header className="mcp-card-header">
          <h2 className="mcp-card-title">
            <span className="mcp-card-icon" aria-hidden="true">🏢</span>
            Account Profile
          </h2>
        </header>
        <div className="mcp-account-profile-empty" aria-label="No account data">
          <span className="mcp-account-profile-empty-icon" aria-hidden="true">🏢</span>
          <p>No account selected</p>
        </div>
      </article>
    );
  }

  const isCached    = account.source === "cache";
  const mrrFormatted = formatCurrency(account.mrrCents);
  const initials    = account.owner
    ? account.owner.split(" ").map((n) => n[0]).join("").toUpperCase().slice(0, 2)
    : "??";

  return (
    <article
      className="mcp-card mcp-account-profile-card"
      aria-label={`Account: ${account.name}`}
      id={`account-card-${account.id}`}
    >
      <header className="mcp-card-header">
        <h2 className="mcp-card-title">
          <span className="mcp-card-icon" aria-hidden="true">🏢</span>
          Account Profile
        </h2>
        {isCached ? (
          <span
            className="mcp-card-badge mcp-card-badge--cached"
            role="status"
            aria-label="Data served from cache snapshot"
            title="Salesforce is unreachable; this data is from the Supabase cache"
          >
            📦 Cached snapshot
          </span>
        ) : (
          <span className="mcp-card-badge mcp-card-badge--live" aria-label="Live data">
            ● Live
          </span>
        )}
      </header>

      {/* Account name and industry */}
      <h3 className="mcp-account-name">{account.name}</h3>
      <span className="mcp-account-industry-tag">{account.industry || "Unknown Industry"}</span>

      {/* Key metrics grid */}
      <dl className="mcp-account-stats-grid">
        <div className="mcp-account-stat">
          <dt className="mcp-account-stat-label">MRR</dt>
          <dd className="mcp-account-stat-value mcp-account-stat-value--green">
            {mrrFormatted}
          </dd>
        </div>
        <div className="mcp-account-stat">
          <dt className="mcp-account-stat-label">Tier</dt>
          <dd className="mcp-account-stat-value mcp-account-stat-value--blue">
            {account.tier || "—"}
          </dd>
        </div>
      </dl>

      {/* Health score progress bar */}
      <div className="mcp-health-score-bar-container">
        <div className="mcp-health-score-bar-label">
          <span>Account Health Score</span>
          <span>{account.healthScore.toFixed(1)}%</span>
        </div>
        <div
          className="mcp-health-score-bar-track"
          role="progressbar"
          aria-valuenow={account.healthScore}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={`Account health score: ${account.healthScore.toFixed(1)}%`}
        >
          <div
            className="mcp-health-score-bar-fill"
            style={{ width: `${Math.min(100, Math.max(0, account.healthScore))}%` }}
          />
        </div>
      </div>

      {/* Account owner */}
      <div className="mcp-account-owner-row">
        <div className="mcp-account-owner-avatar" aria-hidden="true">{initials}</div>
        <div>
          <div className="mcp-account-owner-label">Account Owner</div>
          <div className="mcp-account-owner-name">{account.owner || "Unassigned"}</div>
        </div>
      </div>
    </article>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function formatCurrency(cents: number): string {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(2)}M`;
  if (dollars >= 1_000)     return `$${(dollars / 1_000).toFixed(1)}K`;
  return `$${dollars.toFixed(0)}`;
}
