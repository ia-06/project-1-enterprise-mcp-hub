import { NextRequest, NextResponse } from "next/server";

const GO_RPC_URL = process.env.GO_RPC_URL ?? "http://localhost:8080/rpc";

/**
 * GET /api/health
 *
 * Sends a JSON-RPC 2.0 system.healthCheck call to the Go backend and
 * returns a normalised SystemHealth object to the frontend.
 */
export async function GET(_req: NextRequest) {
  const rpcReq = {
    jsonrpc: "2.0",
    method: "system.healthCheck",
    params: {},
    id: `health-${Date.now()}`,
  };

  try {
    const rpcRes = await fetch(GO_RPC_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rpcReq),
      // Short timeout for health probes — don't wait too long.
      signal: AbortSignal.timeout(5000),
    });

    const body = await rpcRes.json();

    // JSON-RPC error in health check → return degraded status rather than failing.
    if (body.error) {
      return NextResponse.json(
        {
          mcpServer: { status: "down" },
          sales: { status: "unknown" },
          jira: { status: "unknown" },
          salesforce: { status: "unknown", cached: false },
          _rpcError: body.error,
        },
        { status: 200 }
      );
    }

    return NextResponse.json(body.result, { status: 200 });
  } catch (err) {
    // Go server unreachable
    return NextResponse.json(
      {
        mcpServer: { status: "down" },
        sales: { status: "down" },
        jira: { status: "down" },
        salesforce: { status: "down", cached: false },
        _error: err instanceof Error ? err.message : "unknown",
      },
      { status: 200 } // Return 200 so the UI can render a degraded banner.
    );
  }
}
