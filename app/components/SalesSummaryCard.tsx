"use client";

import React from "react";
import type { SalesSummary, SalesOrder } from "@/app/lib/types";

type Props = {
  summary: SalesSummary | null;
  orders: SalesOrder[];
};

/**
 * SalesSummaryCard
 *
 * Renders aggregated sales KPIs (Closed Won, Open Pipeline) and a scrollable
 * list of individual orders with status badges.
 *
 * When summary is null → renders an empty / no-data state.
 */
export default function SalesSummaryCard({ summary, orders }: Props) {
  const hasData = summary !== null;

  return (
    <article
      className="mcp-card mcp-sales-summary-card"
      aria-label="Sales Summary"
      id="sales-summary-card"
    >
      <header className="mcp-card-header">
        <h2 className="mcp-card-title">
          <span className="mcp-card-icon" aria-hidden="true">📈</span>
          Sales Summary
        </h2>
        {hasData && (
          <span className="mcp-card-badge mcp-card-badge--live" aria-label="Postgres live data">
            Postgres
          </span>
        )}
      </header>

      {!hasData ? (
        <div className="mcp-sales-empty" role="status">
          <p>No sales data available</p>
        </div>
      ) : (
        <>
          {/* KPI widgets */}
          <dl className="mcp-sales-kpi-grid">
            <div className="mcp-sales-kpi mcp-sales-kpi--won">
              <dt className="mcp-sales-kpi-label">Closed Won</dt>
              <dd className="mcp-sales-kpi-amount mcp-sales-kpi-amount--won">
                {formatCents(summary!.totalClosedWonCents)}
                <span className="mcp-sales-kpi-currency">USD</span>
              </dd>
            </div>
            <div className="mcp-sales-kpi mcp-sales-kpi--open">
              <dt className="mcp-sales-kpi-label">Open Pipeline</dt>
              <dd className="mcp-sales-kpi-amount mcp-sales-kpi-amount--open">
                {formatCents(summary!.openPipelineCents)}
                <span className="mcp-sales-kpi-currency">USD</span>
              </dd>
            </div>
          </dl>

          {/* Orders list */}
          {orders.length > 0 ? (
            <>
              <p className="mcp-sales-orders-section-title">
                Recent Orders ({orders.length})
              </p>
              <ul className="mcp-sales-orders-list" aria-label="Sales orders">
                {orders.map((order) => (
                  <li key={order.id} className="mcp-sales-order-row">
                    <span className="mcp-sales-order-number">
                      {order.orderNumber}
                    </span>
                    <span className="mcp-sales-order-amount">
                      {formatCents(order.amountCents)}{" "}
                      <span className="mcp-sales-order-number">{order.currency}</span>
                    </span>
                    <span
                      className={`mcp-sales-order-status mcp-sales-order-status--${order.status}`}
                      aria-label={`Status: ${order.status.replace(/_/g, " ").toLowerCase()}`}
                    >
                      {humanizeStatus(order.status)}
                    </span>
                  </li>
                ))}
              </ul>
            </>
          ) : (
            <div className="mcp-sales-empty">
              <p>No orders found for this account</p>
            </div>
          )}
        </>
      )}
    </article>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function formatCents(cents: number): string {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(2)}M`;
  if (dollars >= 1_000)     return `$${(dollars / 1_000).toFixed(1)}K`;
  return `$${dollars.toFixed(0)}`;
}

function humanizeStatus(status: string): string {
  return status.replace(/_/g, " ").toLowerCase().replace(/\b\w/g, (c) => c.toUpperCase());
}
