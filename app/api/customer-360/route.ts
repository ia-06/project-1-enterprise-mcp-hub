import { NextRequest, NextResponse } from "next/server";
import type { Customer360 } from "@/app/lib/types";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/customer-360?accountId=<sf_id>
 *
 * Server-side route that proxies the request to the Go backend's `system.customer360` tool.
 * The Go backend acts as the concurrent fan-out engine, dispatching Goroutines to fetch
 * Salesforce, Jira, and Postgres data simultaneously.
 */
export async function GET(req: NextRequest) {
  const accountId = req.nextUrl.searchParams.get("accountId");

  if (!accountId) {
    return NextResponse.json(
      { code: "MISSING_ACCOUNT_ID", message: "accountId query param is required" },
      { status: 400 }
    );
  }

  const ts = Date.now();
  const rpcReq = {
    jsonrpc: "2.0",
    method: "system.customer360",
    params: { accountId },
    id: `c360-all-${ts}`,
  };

  try {
    const res = await fetch(GO_RPC_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rpcReq),
    });

    const body = await res.json();
    if (body.error) {
      return NextResponse.json(
        { code: "RPC_ERROR", message: body.error.message },
        { status: 500 }
      );
    }

    const customer360 = body.result as Customer360;
    return NextResponse.json(customer360, { status: 200 });

  } catch (error) {
    return NextResponse.json(
      { code: "BACKEND_UNREACHABLE", message: String(error) },
      { status: 502 }
    );
  }
}
