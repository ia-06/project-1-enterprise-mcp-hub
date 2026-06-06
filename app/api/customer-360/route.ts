import { NextRequest, NextResponse } from "next/server";
import type { Account, Customer360, JiraTicket, SalesOrder, SalesSummary } from "@/app/lib/types";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/customer-360?accountId=<sf_id>
 *
 * Server-side aggregation route that fans out three JSON-RPC calls
 * concurrently to the Go backend and assembles a unified Customer360 DTO:
 *
 *   1. salesforce.getAccount  → account data (with Scenario A fallback)
 *   2. sales.listOrders       → orders (uses external_sf_id cross-reference)
 *   3. jira.listTicketsByAccount → linked engineering tickets
 *
 * Partial failures are tolerated: if Jira or Salesforce (cached) fail,
 * the other sections still render.
 */
export async function GET(req: NextRequest) {
  const accountId = req.nextUrl.searchParams.get("accountId");

  if (!accountId) {
    return NextResponse.json(
      { code: "MISSING_ACCOUNT_ID", message: "accountId query param is required" },
      { status: 400 }
    );
  }

  // -------------------------------------------------------------------------
  // Build three JSON-RPC requests
  // -------------------------------------------------------------------------
  const ts = Date.now();

  const accountRPCReq = {
    jsonrpc: "2.0",
    method: "salesforce.getAccount",
    params: { accountId },
    id: `c360-account-${ts}`,
  };

  // For sales: pass the SF Account ID; the Go adapter will map via external_sf_id.
  const salesRPCReq = {
    jsonrpc: "2.0",
    method: "sales.listOrders",
    params: { customerId: "" }, // filled below after account resolution
    id: `c360-sales-${ts}`,
  };

  const ticketsRPCReq = {
    jsonrpc: "2.0",
    method: "jira.listTicketsByAccount",
    params: { accountSfId: accountId },
    id: `c360-tickets-${ts}`,
  };

  // -------------------------------------------------------------------------
  // Fan-out: send account request first to get internal customerId,
  // then fetch sales and tickets concurrently.
  // -------------------------------------------------------------------------
  async function rpcPost(payload: object): Promise<Record<string, unknown>> {
    const res = await fetch(GO_RPC_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    return res.json();
  }

  let account: Account | null = null;
  let salesforceSource: "live" | "cache" | "none" = "none";
  let orders: SalesOrder[] = [];
  let summary: SalesSummary = { totalClosedWonCents: 0, openPipelineCents: 0, orderCount: 0 };
  let tickets: JiraTicket[] = [];
  let jiraMock = true;

  // Step 1: Resolve account (needed to get customerId for sales join).
  try {
    const accountBody = await rpcPost(accountRPCReq);
    if (!accountBody.error) {
      const result = accountBody.result as { account: Account };
      account = result.account;
      salesforceSource = account.source === "cache" ? "cache" : "live";
    }
  } catch {
    // Salesforce down — account stays null; dashboard remains partially functional.
  }

  // Step 2: Fan out sales + tickets concurrently.
  // For sales we pass customerId="" to list all orders when we can't resolve;
  // in production you'd JOIN via external_sf_id. For prototype, pass accountId
  // as customerId param (the Go adapter handles filtering by external_sf_id too).
  salesRPCReq.params.customerId = accountId;

  const [salesBody, ticketsBody] = await Promise.allSettled([
    rpcPost(salesRPCReq),
    rpcPost(ticketsRPCReq),
  ]);

  // Process sales
  if (salesBody.status === "fulfilled" && !salesBody.value.error) {
    const salesResult = salesBody.value.result as { orders: SalesOrder[] };
    orders = salesResult.orders ?? [];

    // Compute summary from orders
    summary = orders.reduce(
      (acc, o) => {
        if (o.status === "CLOSED_WON") acc.totalClosedWonCents += o.amountCents;
        if (o.status === "OPEN" || o.status === "PENDING") acc.openPipelineCents += o.amountCents;
        acc.orderCount++;
        return acc;
      },
      { totalClosedWonCents: 0, openPipelineCents: 0, orderCount: 0 }
    );
  }

  // Process tickets
  if (ticketsBody.status === "fulfilled" && !ticketsBody.value?.error) {
    const ticketsResult = ticketsBody.value.result as { tickets: JiraTicket[] };
    tickets = ticketsResult.tickets ?? [];
    jiraMock = true; // always true in $0-tier dev mode
  }

  // -------------------------------------------------------------------------
  // Assemble and return Customer360 DTO
  // -------------------------------------------------------------------------
  const customer360: Customer360 = {
    account,
    sales: { summary, orders },
    tickets,
    meta: {
      salesforceSource,
      jiraMock,
    },
  };

  return NextResponse.json(customer360, { status: 200 });
}
