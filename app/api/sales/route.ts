import { NextRequest, NextResponse } from "next/server";
import type { SalesOrder } from "@/app/lib/types";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/sales?customerId=<uuid>
 *
 * Proxies to Go JSON-RPC method `sales.listOrders`.
 * Returns a 200 with { orders: SalesOrder[] } on success.
 * Returns 502 with { code: "SALES_DOWN" } when the Postgres adapter fails.
 * Returns 400 with { code: "INVALID_RPC" } for parse/invalid-request errors.
 */
export async function GET(req: NextRequest) {
  const customerId = req.nextUrl.searchParams.get("customerId") ?? "";

  const rpcReq = {
    jsonrpc: "2.0",
    method: "sales.listOrders",
    params: { customerId },
    id: `sales-${Date.now()}`,
  };

  let body: Record<string, unknown>;

  try {
    const rpcRes = await fetch(GO_RPC_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rpcReq),
    });
    body = await rpcRes.json();
  } catch (err) {
    return NextResponse.json(
      { code: "SALES_DOWN", message: "Go backend unreachable", retryable: true },
      { status: 502 }
    );
  }

  if (body.error) {
    const rpcErr = body.error as { code: number; message: string };
    const httpStatus = mapRPCErrorToHTTP(rpcErr.code);
    const errorCode =
      rpcErr.code === -32001 ? "SALES_DOWN" : "INVALID_RPC";
    return NextResponse.json(
      { code: errorCode, rpc: rpcErr, retryable: true },
      { status: httpStatus }
    );
  }

  const result = body.result as { orders: SalesOrder[] };
  return NextResponse.json(result, { status: 200 });
}

// ---------------------------------------------------------------------------
// Shared helper
// ---------------------------------------------------------------------------
function mapRPCErrorToHTTP(code: number): number {
  if (code === -32700 || code === -32600) return 400;
  if (code === -32601) return 404;
  return 502;
}
