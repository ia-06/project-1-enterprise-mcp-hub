"use client";

import React, { useState, useEffect } from "react";
import type { Customer360, SystemHealth, ResilienceMetrics } from "@/app/lib/types";
import { fetchSystemHealth, fetchCustomer360 } from "@/app/lib/api-client";

// Components
import SystemHealthBanner from "@/app/components/SystemHealthBanner";
import AccountProfileCard from "@/app/components/AccountProfileCard";
import SalesSummaryCard from "@/app/components/SalesSummaryCard";
import TicketsList from "@/app/components/TicketsList";
import JsonRpcErrorToast from "@/app/components/JsonRpcErrorToast";

/**
 * Main Customer 360 Dashboard page.
 *
 * State:
 *   - accountId: currently selected Salesforce Account ID
 *   - data: full Customer360 aggregate payload
 *   - health: system health from /api/health probe
 *   - loading: true while fetching customer-360 data
 *   - errorCode: APIErrorCode string when a route handler returns an error
 *   - resilience: derived metrics exposing salesforceStatus and isCachedData
 */
export default function HomePage() {
  const [accountId, setAccountId] = useState<string>("");
  const [data, setData] = useState<Customer360 | null>(null);
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [resilience, setResilience] = useState<ResilienceMetrics>({
    salesforceStatus: "unknown",
    isCachedData: false,
    jiraDegraded: false,
  });

  // Predefined account IDs from seed data for the demo selector
  const demoAccounts = [
    { label: "ACME Corporation",         id: "001ACME000000001" },
    { label: "Beta Technologies Inc.",   id: "001BETA000000002" },
    { label: "Gamma Retail Group",       id: "001GAMA000000003" },
    { label: "Delta Financial Services", id: "001DELT000000004" },
    { label: "Epsilon Healthcare",       id: "001EPSI000000005" },
  ];

  // ------------------------------------------------------------------
  // On mount: probe system health
  // ------------------------------------------------------------------
  useEffect(() => {
    async function loadHealth() {
      try {
        const h = await fetchSystemHealth();
        setHealth(h);
      } catch {
        // Silent health failure — banner will show "System health unknown"
      }
    }
    loadHealth();
  }, []);

  // ------------------------------------------------------------------
  // Derive resilience metrics whenever data changes
  // ------------------------------------------------------------------
  useEffect(() => {
    if (!data) return;

    const sfSource = data.meta?.salesforceSource;
    const isCache = sfSource === "cache";

    setResilience({
      salesforceStatus:
        sfSource === "live"  ? "live"        :
        sfSource === "cache" ? "cache"        :
        sfSource === "none"  ? "unreachable"  : "unknown",
      isCachedData: isCache,
      jiraDegraded: health?.jira?.status === "degraded",
    });
  }, [data, health]);

  // ------------------------------------------------------------------
  // Handle account selection and load Customer 360 data
  // ------------------------------------------------------------------
  async function handleLoad() {
    if (!accountId.trim()) return;
    setLoading(true);
    setErrorCode(null);
    setData(null);

    try {
      const result = await fetchCustomer360(accountId.trim());
      setData(result);
    } catch (err) {
      const code = err instanceof Error ? err.message : "UNKNOWN_ERROR";
      setErrorCode(code);
    } finally {
      setLoading(false);
    }
  }

  function handleAccountSelect(id: string) {
    setAccountId(id);
    setData(null);
    setErrorCode(null);
  }

  return (
    <main className="mcp-dashboard-root">
      {/* ----------------------------------------------------------------
          System Health Banner — always rendered at the top
      ---------------------------------------------------------------- */}
      <SystemHealthBanner health={health} />

      {/* ----------------------------------------------------------------
          Error Toast — shown when route handler returns an error code
      ---------------------------------------------------------------- */}
      <JsonRpcErrorToast
        errorCode={errorCode}
        onClose={() => setErrorCode(null)}
      />

      {/* ----------------------------------------------------------------
          Page Header
      ---------------------------------------------------------------- */}
      <header className="mcp-dashboard-header">
        <div className="mcp-dashboard-header-content">
          <div className="mcp-dashboard-brand">
            <span className="mcp-dashboard-brand-icon" aria-hidden="true">⬡</span>
            <div className="mcp-dashboard-brand-text">
              <h1 className="mcp-dashboard-title">Enterprise MCP Hub</h1>
              <p className="mcp-dashboard-subtitle">Multi-Database Customer 360 Intelligence</p>
            </div>
          </div>

          {/* Resilience status pills */}
          <div className="mcp-resilience-pills" role="status" aria-label="Data source status">
            <span className={`mcp-resilience-pill mcp-resilience-pill--sf-${resilience.salesforceStatus}`}>
              SF: {resilience.salesforceStatus}
            </span>
            {resilience.isCachedData && (
              <span className="mcp-resilience-pill mcp-resilience-pill--cached">
                Cached Data
              </span>
            )}
            {resilience.jiraDegraded && (
              <span className="mcp-resilience-pill mcp-resilience-pill--degraded">
                Jira Degraded
              </span>
            )}
          </div>
        </div>
      </header>

      {/* ----------------------------------------------------------------
          Account Selector
      ---------------------------------------------------------------- */}
      <section className="mcp-account-selector-section" aria-label="Account selection">
        <div className="mcp-account-selector-card">
          <h2 className="mcp-account-selector-heading">Select an Account</h2>
          <p className="mcp-account-selector-description">
            Choose a demo account below or enter a Salesforce Account ID to load a Customer 360 view.
          </p>

          <div className="mcp-account-demo-grid" role="list">
            {demoAccounts.map((account) => (
              <button
                key={account.id}
                className={`mcp-account-demo-button ${accountId === account.id ? "mcp-account-demo-button--selected" : ""}`}
                onClick={() => handleAccountSelect(account.id)}
                role="listitem"
                aria-pressed={accountId === account.id}
                id={`demo-account-${account.id}`}
              >
                <span className="mcp-account-demo-button-name">{account.label}</span>
                <span className="mcp-account-demo-button-id">{account.id}</span>
              </button>
            ))}
          </div>

          <div className="mcp-account-manual-input-row">
            <label className="mcp-account-manual-label" htmlFor="manual-account-id">
              Custom Account ID
            </label>
            <input
              id="manual-account-id"
              className="mcp-account-manual-input"
              type="text"
              placeholder="e.g. 001ACME000000001"
              value={accountId}
              onChange={(e) => setAccountId(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleLoad(); }}
              aria-label="Salesforce Account ID"
            />
            <button
              id="load-customer-360-button"
              className={`mcp-load-button ${loading ? "mcp-load-button--loading" : ""}`}
              onClick={handleLoad}
              disabled={loading || !accountId.trim()}
              aria-label="Load Customer 360"
            >
              {loading ? (
                <span className="mcp-load-button-spinner" aria-hidden="true" />
              ) : null}
              {loading ? "Loading…" : "Load Customer 360"}
            </button>
          </div>
        </div>
      </section>

      {/* ----------------------------------------------------------------
          Customer 360 Dashboard Grid (only rendered when data is available)
      ---------------------------------------------------------------- */}
      {data && (
        <section className="mcp-dashboard-grid-section" aria-label="Customer 360 Dashboard">
          <div className="mcp-dashboard-grid">
            {/* Account profile — spans full width on mobile, 1/3 on desktop */}
            <div className="mcp-dashboard-grid-cell mcp-dashboard-grid-cell--account">
              <AccountProfileCard account={data.account} />
            </div>

            {/* Sales summary */}
            <div className="mcp-dashboard-grid-cell mcp-dashboard-grid-cell--sales">
              <SalesSummaryCard
                summary={data.sales?.summary ?? null}
                orders={data.sales?.orders ?? []}
              />
            </div>

            {/* Jira tickets — full width */}
            <div className="mcp-dashboard-grid-cell mcp-dashboard-grid-cell--tickets">
              <TicketsList
                tickets={data.tickets ?? []}
                jiraDegraded={resilience.jiraDegraded}
              />
            </div>
          </div>

          {/* Metadata footer */}
          <footer className="mcp-dashboard-meta-footer" aria-label="Data source metadata">
            <span className="mcp-meta-badge">
              Salesforce: <strong>{data.meta?.salesforceSource}</strong>
            </span>
            <span className="mcp-meta-badge">
              Jira: <strong>{data.meta?.jiraMock ? "mock" : "live"}</strong>
            </span>
            <span className="mcp-meta-badge">
              Orders: <strong>{data.sales?.orders?.length ?? 0}</strong>
            </span>
          </footer>
        </section>
      )}

      {/* Empty state when no account is selected and no data loaded */}
      {!data && !loading && !errorCode && (
        <section className="mcp-empty-state" aria-label="Empty state">
          <div className="mcp-empty-state-content">
            <span className="mcp-empty-state-icon" aria-hidden="true">⬡</span>
            <h2 className="mcp-empty-state-heading">Select an account to begin</h2>
            <p className="mcp-empty-state-body">
              Choose a demo account above or enter a Salesforce Account ID to load the
              unified Customer 360 intelligence view.
            </p>
          </div>
        </section>
      )}
    </main>
  );
}
