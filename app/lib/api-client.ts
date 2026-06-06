/**
 * api-client.ts
 *
 * Client-side fetch wrapper for all /api/* route handlers.
 * Provides typed, error-normalised access to the MCP hub data layer.
 * All functions throw an Error with an APIErrorCode message on failure,
 * allowing calling components to pattern-match on specific error states.
 */

import type { Account, Customer360, JiraTicket, SalesOrder, SystemHealth, APIErrorCode } from "./types";

// ---------------------------------------------------------------------------
// Internal fetch helper
// ---------------------------------------------------------------------------

/**
 * handleResponse reads the response body and either returns the typed payload
 * or throws an Error whose message is the APIErrorCode string.
 */
async function handleResponse<T>(res: Response): Promise<T> {
  let body: Record<string, unknown> | null = null;
  try {
    body = await res.json();
  } catch {
    throw new Error("UNKNOWN_ERROR" satisfies APIErrorCode);
  }

  if (!res.ok) {
    const code = (body?.code as APIErrorCode) ?? "UNKNOWN_ERROR";
    throw new Error(code);
  }

  return body as T;
}

// ---------------------------------------------------------------------------
// Public API surface
// ---------------------------------------------------------------------------

/**
 * fetchSystemHealth — probes all backing services and returns their health status.
 * Used by SystemHealthBanner and the top-level page on mount.
 */
export async function fetchSystemHealth(): Promise<SystemHealth> {
  const res = await fetch("/api/health", {
    method: "GET",
    cache: "no-store",
  });
  return handleResponse<SystemHealth>(res);
}

/**
 * fetchCustomer360 — aggregates account, sales, and ticket data for a given
 * Salesforce Account ID. This is the primary data-fetching call for the dashboard.
 */
export async function fetchCustomer360(accountId: string): Promise<Customer360> {
  const res = await fetch(
    `/api/customer-360?accountId=${encodeURIComponent(accountId)}`,
    { cache: "no-store" }
  );
  return handleResponse<Customer360>(res);
}

/**
 * fetchAccount — fetches a single Salesforce Account (with fallback to cache).
 */
export async function fetchAccount(accountId: string): Promise<Account> {
  const res = await fetch(
    `/api/accounts?accountId=${encodeURIComponent(accountId)}`,
    { cache: "no-store" }
  );
  return handleResponse<Account>(res);
}

/**
 * fetchSalesOrders — fetches sales orders, optionally filtered by customer UUID.
 */
export async function fetchSalesOrders(customerId?: string): Promise<SalesOrder[]> {
  const params = customerId
    ? `?customerId=${encodeURIComponent(customerId)}`
    : "";
  const res = await fetch(`/api/sales${params}`, { cache: "no-store" });
  const data = await handleResponse<{ orders: SalesOrder[] }>(res);
  return data.orders ?? [];
}

/**
 * fetchTickets — fetches Jira tickets linked to a Salesforce Account ID.
 */
export async function fetchTickets(accountSfId: string): Promise<JiraTicket[]> {
  const res = await fetch(
    `/api/tickets?accountSfId=${encodeURIComponent(accountSfId)}`,
    { cache: "no-store" }
  );
  const data = await handleResponse<{ tickets: JiraTicket[] }>(res);
  return data.tickets ?? [];
}
