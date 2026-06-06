"use client";

import React from "react";
import type { JiraTicket } from "@/app/lib/types";

type Props = {
  tickets: JiraTicket[];
  jiraDegraded: boolean;
};

/**
 * TicketsList
 *
 * Renders a tabular list of Jira engineering tickets linked to the selected account.
 *
 * Resilience behaviors:
 *   - jiraDegraded=true   → shows a subtle yellow warning bar above the table
 *   - tickets.length === 0 → shows empty state message
 */
export default function TicketsList({ tickets, jiraDegraded }: Props) {
  return (
    <article
      className="mcp-card mcp-tickets-card"
      aria-label="Jira Engineering Tickets"
      id="tickets-list-card"
    >
      <header className="mcp-card-header">
        <h2 className="mcp-card-title">
          <span className="mcp-card-icon" aria-hidden="true">🎫</span>
          Engineering Tickets
        </h2>
        <span className="mcp-card-badge mcp-card-badge--mock" aria-label="Jira data source">
          Jira
        </span>
      </header>

      {/* Degraded warning banner */}
      {jiraDegraded && (
        <div
          className="mcp-tickets-degraded-warning"
          role="alert"
          aria-live="polite"
        >
          <span aria-hidden="true">⚠</span>
          Jira is running in degraded mode. Ticket data may be stale or incomplete.
        </div>
      )}

      {tickets.length === 0 ? (
        <div className="mcp-tickets-empty" role="status">
          <p>No active tickets linked to this account</p>
        </div>
      ) : (
        <div className="mcp-tickets-table-wrapper">
          <table
            className="mcp-tickets-table"
            aria-label={`${tickets.length} engineering tickets`}
          >
            <thead>
              <tr>
                <th scope="col">Key</th>
                <th scope="col">Summary</th>
                <th scope="col">Priority</th>
                <th scope="col">Status</th>
                <th scope="col">Assignee</th>
                <th scope="col">Updated</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map((ticket) => (
                <tr key={ticket.key} id={`ticket-row-${ticket.key}`}>
                  <td>
                    <span className="mcp-ticket-key">{ticket.key}</span>
                  </td>
                  <td>
                    <span className="mcp-ticket-summary">{ticket.summary}</span>
                  </td>
                  <td>
                    <span
                      className={`mcp-ticket-priority mcp-ticket-priority--${ticket.priority}`}
                      aria-label={`Priority: ${ticket.priority}`}
                    >
                      {ticket.priority}
                    </span>
                  </td>
                  <td>
                    <span
                      className={`mcp-ticket-status mcp-ticket-status--${slugifyStatus(ticket.status)}`}
                      aria-label={`Status: ${ticket.status}`}
                    >
                      {ticket.status}
                    </span>
                  </td>
                  <td>{ticket.assignee || "Unassigned"}</td>
                  <td>{formatDate(ticket.updatedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </article>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function slugifyStatus(status: string): string {
  return status.toLowerCase().replace(/\s+/g, "-");
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}
