// Shared TypeScript type definitions used across Next.js route handlers,
// React components, and client-side data-fetching hooks.
// These types mirror the Go domain structs in mcp-server/internal/domain/models.go.

// ---------------------------------------------------------------------------
// Salesforce Account
// ---------------------------------------------------------------------------
export type Account = {
  id: string;
  name: string;
  tier: string;
  mrrCents: number;
  healthScore: number;
  owner: string;
  industry: string;
  /** "live" when fetched from Salesforce directly; "cache" when served from
   *  the Supabase fallback (Scenario A). */
  source?: "live" | "cache";
};

// ---------------------------------------------------------------------------
// Sales data
// ---------------------------------------------------------------------------
export type SalesOrder = {
  id: string;
  customerId: string;
  orderNumber: string;
  amountCents: number;
  currency: string;
  status: "OPEN" | "CLOSED_WON" | "CLOSED_LOST" | "PENDING";
  closedAt?: string;
  createdAt: string;
};

export type SalesSummary = {
  totalClosedWonCents: number;
  openPipelineCents: number;
  orderCount: number;
};

// ---------------------------------------------------------------------------
// Jira Tickets
// ---------------------------------------------------------------------------
export type JiraTicket = {
  key: string;
  summary: string;
  status: string;
  assignee: string;
  priority: string;
  updatedAt: string;
  accountSfId?: string;
};

// ---------------------------------------------------------------------------
// Aggregated Customer 360 view
// ---------------------------------------------------------------------------
export type Customer360 = {
  account: Account | null;
  sales: {
    summary: SalesSummary;
    orders: SalesOrder[];
  };
  tickets: JiraTicket[];
  meta: {
    /** "live" | "cache" | "none" — indicates Salesforce data source used. */
    salesforceSource: "live" | "cache" | "none";
    /** true when Jira is running in mock mode */
    jiraMock: boolean;
  };
};

// ---------------------------------------------------------------------------
// System health
// ---------------------------------------------------------------------------
export type ServiceStatus = "up" | "degraded" | "down";

export type SystemHealth = {
  mcpServer: { status: ServiceStatus };
  sales: { status: ServiceStatus };
  jira: { status: ServiceStatus };
  salesforce: { status: ServiceStatus; cached: boolean };
};

// ---------------------------------------------------------------------------
// API error codes surfaced from route handlers
// ---------------------------------------------------------------------------
export type APIErrorCode =
  | "MISSING_ACCOUNT_ID"
  | "MISSING_CUSTOMER_ID"
  | "SALESFORCE_ERROR"
  | "SALESFORCE_DOWN"
  | "SALES_DOWN"
  | "JIRA_DOWN"
  | "INVALID_RPC"
  | "RPC_METHOD_NOT_FOUND"
  | "UNKNOWN_ERROR";

export type APIError = {
  code: APIErrorCode;
  message?: string;
  retryable?: boolean;
};

// ---------------------------------------------------------------------------
// Resilience state exposed by the useCustomer360 hook
// ---------------------------------------------------------------------------
export type ResilienceMetrics = {
  salesforceStatus: "live" | "cache" | "unreachable" | "unknown";
  isCachedData: boolean;
  jiraDegraded: boolean;
};
