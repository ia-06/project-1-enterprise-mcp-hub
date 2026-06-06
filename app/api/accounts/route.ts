import { NextRequest, NextResponse } from "next/server";
import type { Account } from "@/app/lib/types";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/accounts?accountId=<sf_id>
 *
 * Proxies to Go JSON-RPC method `salesforce.getAccount`.
 *
 * System Resilience Matrix — Scenario A (Salesforce CRM Unreachable):
 *   - When Go returns error code -32002 with data.fallback="supabase",
 *     the account data was already resolved via the Supabase cache at the
 *     Go layer. The result will contain account.source="cache".
 *   - When Go returns a plain -32002 with no cached payload:
 *     → HTTP 502 + { code: "SALESFORCE_DOWN", retryable: true }
 *
 * Scenario B (Invalid JSON-RPC packet):
 *   - -32700 / -32600 → HTTP 400 + { code: "INVALID_RPC" }
 *   - -32601           → HTTP 404 + { code: "RPC_METHOD_NOT_FOUND" }
 */
export async function GET(req: NextRequest) {
  const accountId = req.nextUrl.searchParams.get("accountId");

  if (!accountId) {
    return NextResponse.json(
      { code: "MISSING_ACCOUNT_ID", message: "accountId query param is required" },
      { status: 400 }
    );
  }

  const rpcReq = {
    jsonrpc: "2.0",
    method: "salesforce.getAccount",
    params: { accountId },
    id: `accounts-${Date.now()}`,
  };

  let body: Record<string, unknown>;

  try {
    const rpcRes = await fetch(GO_RPC_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rpcReq),
    });
    body = await rpcRes.json();
  } catch {
    return NextResponse.json(
      { code: "SALESFORCE_DOWN", message: "Go backend unreachable", retryable: true },
      { status: 502 }
    );
  }

  // -------------------------------------------------------------------------
  // Handle JSON-RPC errors (Scenario A & B)
  // -------------------------------------------------------------------------
  if (body.error) {
    const rpcErr = body.error as {
      code: number;
      message: string;
      data?: { fallback?: string; retryable?: boolean };
    };

    // Scenario B: Parse / Invalid request errors
    if (rpcErr.code === -32700 || rpcErr.code === -32600) {
      return NextResponse.json({ code: "INVALID_RPC", rpc: rpcErr }, { status: 400 });
    }
    if (rpcErr.code === -32601) {
      return NextResponse.json({ code: "RPC_METHOD_NOT_FOUND", rpc: rpcErr }, { status: 404 });
    }

    // Scenario A: Salesforce unreachable, no cache hit
    return NextResponse.json(
      {
        code: "SALESFORCE_DOWN",
        rpc: rpcErr,
        retryable: rpcErr.data?.retryable ?? true,
        resiliency: { salesforce: "unreachable" },
      },
      { status: 502 }
    );
  }

  // -------------------------------------------------------------------------
  // Success — may include source="cache" for the Scenario A cache-hit path
  // -------------------------------------------------------------------------
  const result = body.result as { account: Account };
  const account = result.account;

  return NextResponse.json(
    {
      ...account,
      // Expose resiliency metadata so the frontend can show the cache badge.
      resiliency: account.source === "cache"
        ? { salesforce: "unreachable" }
        : { salesforce: "live" },
    },
    { status: 200 }
  );
}
