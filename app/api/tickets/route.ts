import { NextRequest, NextResponse } from "next/server";
import type { JiraTicket } from "@/app/lib/types";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/tickets?accountSfId=<sf_id>
 *
 * Proxies to Go JSON-RPC method `jira.listTicketsByAccount`.
 * Returns 400 if accountSfId is missing.
 * Returns 200 with { tickets: JiraTicket[] } on success.
 * Returns 502 with { code: "JIRA_DOWN" } on backend failure.
 */
export async function GET(req: NextRequest) {
  const accountSfId = req.nextUrl.searchParams.get("accountSfId");

  if (!accountSfId) {
    return NextResponse.json(
      { code: "MISSING_ACCOUNT_ID", message: "accountSfId query param is required" },
      { status: 400 }
    );
  }

  const rpcReq = {
    jsonrpc: "2.0",
    method: "jira.listTicketsByAccount",
    params: { accountSfId },
    id: `tickets-${Date.now()}`,
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
      { code: "JIRA_DOWN", message: "Go backend unreachable", retryable: true },
      { status: 502 }
    );
  }

  if (body.error) {
    const rpcErr = body.error as { code: number; message: string };
    const httpStatus = mapRPCErrorToHTTP(rpcErr.code);
    return NextResponse.json(
      { code: "JIRA_DOWN", rpc: rpcErr, retryable: true },
      { status: httpStatus }
    );
  }

  const result = body.result as { tickets: JiraTicket[] };
  return NextResponse.json(result, { status: 200 });
}

function mapRPCErrorToHTTP(code: number): number {
  if (code === -32700 || code === -32600) return 400;
  if (code === -32601) return 404;
  return 502;
}
