"use client";

import { useState, useEffect, useCallback, useRef } from "react";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface Account {
  id: string;
  name: string;
  tier?: string;
  mrrCents?: number;
  healthScore?: number;
  owner?: string;
  industry?: string;
  source?: string;
}

interface JiraTicket {
  key: string;
  summary: string;
  status: string;
  assignee: string;
  priority: string;
  updatedAt: string;
  accountSfId?: string;
}

interface SalesOrder {
  id: string;
  customerId: string;
  orderNumber: string;
  amountCents: number;
  currency: string;
  status: string;
  closedAt?: string;
  createdAt: string;
}

interface HealthStatus {
  status: "up" | "degraded" | "down";
  cached?: boolean;
}

interface SystemHealth {
  mcpServer: HealthStatus;
  sales:     HealthStatus;
  jira:      HealthStatus;
  salesforce: HealthStatus;
}

type Tab = "account" | "tickets" | "orders";

// ---------------------------------------------------------------------------
// RPC helper
// ---------------------------------------------------------------------------
async function rpc<T>(method: string, params: object = {}): Promise<T> {
  const res = await fetch(process.env.NEXT_PUBLIC_GO_RPC_URL || "http://localhost:8080/rpc", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", method, params, id: Date.now() }),
  });
  const data = await res.json();
  if (data.error) throw new Error(`[${data.error.code}] ${data.error.message}`);
  return data.result as T;
}

// ---------------------------------------------------------------------------
// Formatters
// ---------------------------------------------------------------------------
const fmt = {
  currency: (cents: number, curr = "USD") =>
    new Intl.NumberFormat("en-US", { style: "currency", currency: curr, maximumFractionDigits: 0 }).format(cents / 100),
  date: (iso: string) =>
    iso ? new Date(iso).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }) : "—",
  scoreColor: (s: number) => s >= 70 ? "#2da562" : s >= 40 ? "#f5a623" : "#ee0000",
};

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------
function StatusDot({ status }: { status: string }) {
  const cls = status === "up" ? "health-dot-up" : status === "degraded" ? "health-dot-degraded" : "health-dot-down";
  return <span className={`health-dot ${cls}`} />;
}

function Skeleton({ w = "100%", h = "14px" }: { w?: string; h?: string }) {
  return <div className="skeleton" style={{ width: w, height: h, borderRadius: "4px" }} />;
}

function PriorityBadge({ p }: { p: string }) {
  const map: Record<string, string> = {
    Highest: "badge-error", High: "badge-error", Medium: "badge-warning",
    Low: "badge", Lowest: "badge",
  };
  return <span className={`badge ${map[p] || "badge"}`}>{p || "—"}</span>;
}

function StatusBadge({ s }: { s: string }) {
  const map: Record<string, string> = {
    CLOSED_WON: "badge-success", OPEN: "badge-blue", CLOSED_LOST: "badge-error",
  };
  return <span className={`badge ${map[s] || "badge"}`}>{s.replace("_", " ")}</span>;
}

// ---------------------------------------------------------------------------
// Global Cache (Persists across unmounts in the same session)
// ---------------------------------------------------------------------------
const cache = {
  health: null as SystemHealth | null,
  accounts: null as Account[] | null,
  selectedId: null as string | null,
  details: {} as Record<string, Account>,
  tickets: {} as Record<string, JiraTicket[]>,
  orders: {} as Record<string, SalesOrder[]>,
};

const HEALTH_KEYS = ["mcpServer", "jira", "salesforce", "sales"] as const;

// ---------------------------------------------------------------------------
// Dashboard page
// ---------------------------------------------------------------------------
export default function DashboardPage() {
  // -- State --
  const [accounts, setAccounts]         = useState<Account[]>(cache.accounts || []);
  const [selected, setSelected]         = useState<Account | null>(() => {
    if (cache.selectedId && cache.accounts) {
      return cache.accounts.find((a) => a.id === cache.selectedId) || cache.accounts[0];
    }
    return cache.accounts?.[0] || null;
  });
  const [health, setHealth]             = useState<SystemHealth | null>(cache.health);
  const [tab, setTab]                   = useState<Tab>("account");
  
  const [accountDetail, setAccountDetail] = useState<Account | null>(() => selected ? cache.details[selected.id] || null : null);
  const [tickets, setTickets]           = useState<JiraTicket[]>(() => selected ? cache.tickets[selected.id] || [] : []);
  const [orders, setOrders]             = useState<SalesOrder[]>(() => selected ? cache.orders[selected.id] || [] : []);
  
  const [loadingAccounts, setLoadingAccounts] = useState(!cache.accounts);
  const [loadingDetail, setLoadingDetail]     = useState(false);
  const [error, setError]               = useState<string | null>(null);
  
  // -- Search State --
  const [searchQuery, setSearchQuery] = useState("");
  const [isSearching, setIsSearching] = useState(false);

  // -- Seeder State --
  const [showSeedModal, setShowSeedModal] = useState(false);
  const [isSeeding, setIsSeeding] = useState(false);
  const [hasPromptedSeed, setHasPromptedSeed] = useState(false);
  const [seedProgress, setSeedProgress] = useState<{ total: number; completed: number; isComplete: boolean; logs: string[] } | null>(null);
  const logsEndRef = useRef<HTMLDivElement>(null);

  // -- Load health + account list on mount --
  useEffect(() => {
    if (cache.accounts) return; // already loaded

    Promise.all([
      fetch((process.env.NEXT_PUBLIC_GO_RPC_URL || "http://localhost:8080/rpc").replace("/rpc", "/health"))
        .then((r) => r.json())
        .catch(() => null),
      rpc<{ accounts: Account[] }>("salesforce.listAccounts", { limit: 50 }),
    ])
      .then(([h, accs]) => {
        if (h) {
          setHealth(h);
          cache.health = h;
        }
        const list = accs?.accounts ?? [];
        setAccounts(list);
        cache.accounts = list;
        
        if (list.length > 0) {
          setSelected(list[0]);
          cache.selectedId = list[0].id;
        }
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoadingAccounts(false));
  }, []);

  // -- Search effect --
  useEffect(() => {
    if (!searchQuery.trim()) {
      if (cache.accounts && cache.accounts.length > 0 && accounts.length !== cache.accounts.length) {
        setAccounts(cache.accounts);
      }
      return;
    }

    const delay = setTimeout(async () => {
      setIsSearching(true);
      try {
        const res = await rpc<{ accounts: Account[] }>("salesforce.searchAccounts", { query: searchQuery });
        setAccounts(res.accounts ?? []);
      } catch (e) {
        // silent fail for search
      } finally {
        setIsSearching(false);
      }
    }, 500);

    return () => clearTimeout(delay);
  }, [searchQuery, accounts.length]);

  // -- Load account detail / tickets / orders when selection changes --
  const loadDetail = useCallback(async (acc: Account, force?: boolean) => {
    cache.selectedId = acc.id;
    if (!force && cache.details[acc.id] && cache.tickets[acc.id] && cache.orders[acc.id]) {
      setAccountDetail(cache.details[acc.id]);
      setTickets(cache.tickets[acc.id]);
      setOrders(cache.orders[acc.id] || []);
      return;
    }

    setLoadingDetail(true);
    setAccountDetail(null);
    setTickets([]);
    setOrders([]);
    setError(null);

    try {
      // Unified fetch via system.customer360 to get Account, Tickets, and Orders in one round trip
      const c360 = await rpc<{ account: Account, tickets: JiraTicket[], sales: { orders: SalesOrder[] } }>(
        "system.customer360", 
        { accountId: acc.id }
      );
      
      const fetchedAccount = c360.account;
      const fetchedTickets = c360.tickets ?? [];
      const fetchedOrders = c360.sales?.orders ?? [];

      setAccountDetail(fetchedAccount);
      setTickets(fetchedTickets);
      setOrders(fetchedOrders);
      
      // Inject all into cache simultaneously
      cache.details[acc.id] = fetchedAccount;
      cache.tickets[acc.id] = fetchedTickets;
      cache.orders[acc.id] = fetchedOrders;

      // Auto-prompt seeding if no orders exist
      if (fetchedOrders.length === 0 && !hasPromptedSeed) {
        setShowSeedModal(true);
        setHasPromptedSeed(true);
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load unified account data");
    } finally {
      setLoadingDetail(false);
    }
  }, [hasPromptedSeed]);

  useEffect(() => {
    if (selected) {
      loadDetail(selected);
      setTab("account");
    }
  }, [selected, loadDetail]);



  // Handle manual seed trigger
  const handleSeedData = async () => {
    setIsSeeding(true);
    setSeedProgress({ total: 100, completed: 0, isComplete: false, logs: ["Connecting to MCP server..."] }); // Optimistic init

    try {
      await fetch((process.env.NEXT_PUBLIC_GO_RPC_URL || "http://localhost:8080/rpc").replace("/rpc", "/api/seed"), {
        method: "POST",
      });
      
      // Start polling
      const poll = async () => {
        try {
          const res = await fetch((process.env.NEXT_PUBLIC_GO_RPC_URL || "http://localhost:8080/rpc").replace("/rpc", "/api/seed/status"));
          const data = await res.json();
          setSeedProgress({
            total: data.totalAccounts,
            completed: data.completedAccounts,
            isComplete: data.isComplete,
            logs: data.logs || []
          });
          
          if (data.isComplete) {
            setIsSeeding(false);
            setShowSeedModal(false);
            setSeedProgress(null);
            
            // Invalidate cache and auto-refresh the current view
            cache.details = {};
            cache.tickets = {};
            cache.orders = {};
            if (selected) loadDetail(selected, true);
          } else {
            setTimeout(poll, 1000);
          }
        } catch (e) {
          setTimeout(poll, 2000);
        }
      };
      
      poll();

    } catch (e) {
      alert("Failed to start seeding.");
      setIsSeeding(false);
      setSeedProgress(null);
    }
  };

  // Auto-scroll logs
  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [seedProgress?.logs]);

  // -- Render --
  return (
    <>
      {/* Health banner */}
      {health && (
        <div className="health-banner">
          <span className="caption-mono" style={{ marginRight: "var(--space-sm)" }}>
            System Status
          </span>
          {HEALTH_KEYS.map((key, i) => {
            const labels: Record<string, string> = {
              mcpServer: "MCP Server", jira: "Jira", salesforce: "Salesforce", sales: "Postgres",
            };
            return (
              <span key={key} style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                {i > 0 && <span className="health-divider" />}
                <StatusDot status={health[key].status} />
                <span>{labels[key]}</span>
                {health[key].cached && (
                  <span className="badge badge-warning" style={{ fontSize: "10px" }}>cached</span>
                )}
              </span>
            );
          })}
        </div>
      )}

      <div className="dashboard-layout">
        {/* ── Sidebar ── */}
        <aside className="dashboard-sidebar">
          <div className="sidebar-header" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div>
              <div className="sidebar-label">Salesforce Accounts</div>
              <p className="caption" style={{ color: "var(--mute)" }}>
                {loadingAccounts ? "Loading..." : `${accounts.length} accounts`}
              </p>
            </div>
            <button 
              className="btn btn-secondary" 
              style={{ height: "28px", padding: "0 10px", fontSize: "11px" }}
              onClick={() => setShowSeedModal(true)}
            >
              Populate Data
            </button>
          </div>
          
          <div style={{ padding: "0 var(--space-md) var(--space-md) var(--space-md)", borderBottom: "1px solid var(--hairline)" }}>
            <div style={{ 
              display: "flex", 
              alignItems: "center",
              background: "var(--canvas)",
              border: "1px solid var(--hairline)",
              borderRadius: "var(--r-sm)",
              padding: "6px 12px",
              boxShadow: "0 1px 2px rgba(0,0,0,0.02)",
              transition: "border-color 0.15s ease"
            }}>
              <svg 
                width="14" 
                height="14" 
                viewBox="0 0 24 24" 
                fill="none" 
                stroke="currentColor" 
                strokeWidth="2" 
                strokeLinecap="round" 
                strokeLinejoin="round" 
                style={{ color: "var(--mute)", flexShrink: 0, marginTop: "1px" }}
              >
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <input 
                type="text" 
                placeholder="Search accounts..." 
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                style={{ 
                  width: "100%", 
                  fontSize: "13px", 
                  background: "transparent", 
                  border: "none", 
                  outline: "none",
                  marginLeft: "8px",
                  color: "var(--ink)",
                  fontFamily: "inherit"
                }}
              />
            </div>
          </div>

          <div className="account-list">
            {loadingAccounts || isSearching ? (
              Array.from({ length: 5 }).map((_, i) => (
                <div key={i} style={{ padding: "var(--space-sm)", display: "flex", flexDirection: "column", gap: "6px" }}>
                  <Skeleton h="14px" w="80%" />
                  <Skeleton h="10px" w="50%" />
                </div>
              ))
            ) : accounts.length === 0 ? (
              <div style={{ padding: "var(--space-md)", textAlign: "center" }}>
                <p className="caption" style={{ color: "var(--mute)" }}>No accounts found</p>
              </div>
            ) : (
              accounts.map((acc) => (
                <button
                  key={acc.id}
                  className={`account-item${selected?.id === acc.id ? " selected" : ""}`}
                  onClick={() => setSelected(acc)}
                >
                  <span className="account-item-name">{acc.name}</span>
                  <span className="account-item-meta" style={{ display: "flex", alignItems: "center", gap: "6px", flexWrap: "wrap", marginTop: "2px" }}>
                    {acc.industry && <span>{acc.industry}</span>}
                    {acc.industry && acc.tier && <span style={{ opacity: 0.5 }}>•</span>}
                    {acc.tier && <span>{acc.tier}</span>}
                    {(acc.industry || acc.tier) && <span style={{ opacity: 0.5 }}>•</span>}
                    <span style={{ fontFamily: "var(--font-mono)", fontSize: "10px", color: "var(--mute)" }}>
                      {acc.id}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>
        </aside>

        {/* ── Main panel ── */}
        <main className="dashboard-main">
          {!selected ? (
            <div className="empty-state">
              <h3>Select an account</h3>
              <p>Choose a Salesforce account from the sidebar to view the Customer 360 panel.</p>
            </div>
          ) : (
            <>
              {/* Account header */}
              <div style={{ marginBottom: "var(--space-lg)" }}>
                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: "var(--space-sm)" }}>
                  <div>
                    <h1 className="display-md" style={{ color: "var(--ink)" }}>
                      {loadingDetail ? <Skeleton w="200px" h="28px" /> : (accountDetail?.name ?? selected.name)}
                    </h1>
                    <div style={{ display: "flex", alignItems: "center", gap: "var(--space-xs)", marginTop: "var(--space-xs)" }}>
                      {accountDetail?.tier && <span className="badge">{accountDetail.tier}</span>}
                      {accountDetail?.industry && <span className="badge">{accountDetail.industry}</span>}
                      {accountDetail?.source === "cache" && (
                        <span className="badge badge-warning" title="Salesforce unreachable — showing cached data">
                          ⚡ Cached
                        </span>
                      )}
                      <span className="caption-mono" style={{ color: "var(--mute)" }}>
                        {selected.id}
                      </span>
                    </div>
                  </div>
                  <button
                    className="btn btn-secondary btn-sm"
                    onClick={() => loadDetail(selected, true)}
                    disabled={loadingDetail}
                  >
                    {loadingDetail ? "Refreshing…" : "↻ Refresh"}
                  </button>
                </div>

                {error && (
                  <div
                    style={{
                      marginTop: "var(--space-md)",
                      background: "var(--error-soft)",
                      border: "1px solid var(--error)",
                      borderRadius: "var(--r-sm)",
                      padding: "var(--space-sm) var(--space-md)",
                      fontSize: "13px",
                      color: "var(--error)",
                    }}
                  >
                    {error}
                  </div>
                )}
              </div>

              {/* Stat cards */}
              <div className="stat-grid">
                {[
                  {
                    label: "Health Score",
                    value: loadingDetail ? null : accountDetail?.healthScore != null
                      ? `${accountDetail.healthScore.toFixed(0)}%`
                      : "—",
                    sub: "Account Health",
                    color: accountDetail?.healthScore != null ? fmt.scoreColor(accountDetail.healthScore) : undefined,
                  },
                  {
                    label: "MRR",
                    value: loadingDetail ? null : accountDetail?.mrrCents != null
                      ? fmt.currency(accountDetail.mrrCents)
                      : "—",
                    sub: "Monthly Recurring",
                  },
                  {
                    label: "Open Tickets",
                    value: loadingDetail ? null : tickets.filter((t) => t.status?.toLowerCase() !== "done").length.toString(),
                    sub: "Jira Backlog",
                  },
                ].map((s) => (
                  <div key={s.label} className="stat-card">
                    <div className="stat-label">{s.label}</div>
                    {s.value === null ? (
                      <Skeleton w="80px" h="28px" />
                    ) : (
                      <div className="stat-value" style={s.color ? { color: s.color } : {}}>
                        {s.value}
                      </div>
                    )}
                    <div className="stat-sub">{s.sub}</div>
                  </div>
                ))}
              </div>

              {/* Tabs */}
              <div className="tab-row" style={{ marginTop: "var(--space-lg)" }}>
                {(["account", "tickets", "orders"] as Tab[]).map((t) => (
                  <button
                    key={t}
                    className={`tab${tab === t ? " active" : ""}`}
                    onClick={() => setTab(t)}
                  >
                    {t === "account" ? "Account" : t === "tickets" ? `Tickets (${tickets.length})` : `Orders (${orders.length})`}
                  </button>
                ))}
              </div>

              {/* Tab content */}
              {tab === "account" && (
                <div className="card">
                  {loadingDetail ? (
                    <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-sm)" }}>
                      {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} h="18px" />)}
                    </div>
                  ) : accountDetail ? (
                    <dl style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "var(--space-md) var(--space-xl)" }}>
                      {[
                        ["Salesforce ID",  accountDetail.id],
                        ["Account Name",   accountDetail.name],
                        ["Type / Tier",    accountDetail.tier || "—"],
                        ["Industry",       accountDetail.industry || "—"],
                        ["Account Owner",  accountDetail.owner || "—"],
                        ["Data Source",    accountDetail.source === "cache" ? "⚡ Supabase Cache" : "✓ Live Salesforce"],
                      ].map(([k, v]) => (
                        <div key={k}>
                          <dt className="caption-mono" style={{ color: "var(--mute)", marginBottom: "4px" }}>{k}</dt>
                          <dd style={{ fontSize: "14px", color: "var(--ink)", fontWeight: 500 }}>{v}</dd>
                        </div>
                      ))}
                    </dl>
                  ) : (
                    <div className="empty-state">
                      <h3>No account data</h3>
                      <p>Salesforce may be unreachable and the Supabase cache returned no match.</p>
                    </div>
                  )}
                </div>
              )}

              {tab === "tickets" && (
                <div>
                  {loadingDetail ? (
                    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--space-sm)" }}>
                      {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} h="18px" />)}
                    </div>
                  ) : tickets.length === 0 ? (
                    <div className="empty-state">
                      <h3>No tickets found.</h3>
                      <p>
                        Make sure the Jira issue custom field{" "}
                        <code style={{ fontFamily: "monospace", fontSize: "12px" }}>Salesforce Account ID</code>{" "}
                        is set to <strong>{selected.id}</strong>.
                      </p>
                    </div>
                  ) : (
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>Key</th>
                          <th>Summary</th>
                          <th>Status</th>
                          <th>Assignee</th>
                          <th>Priority</th>
                          <th>Updated</th>
                        </tr>
                      </thead>
                      <tbody>
                        {tickets.map((t) => (
                          <tr key={t.key}>
                            <td style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px", color: "var(--link)", whiteSpace: "nowrap" }}>
                              {t.key}
                            </td>
                            <td style={{ color: "var(--ink)", fontWeight: 500 }}>{t.summary}</td>
                            <td><span className="badge">{t.status}</span></td>
                            <td>{t.assignee || "—"}</td>
                            <td><PriorityBadge p={t.priority} /></td>
                            <td style={{ whiteSpace: "nowrap", color: "var(--mute)" }}>{fmt.date(t.updatedAt)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              )}

              {tab === "orders" && (
                <div>
                  {orders.length === 0 ? (
                    <div className="empty-state">
                      <h3>No orders loaded.</h3>
                      <p>Sales orders are linked to internal Postgres customer UUIDs. Map your SF Account ID to a customer record.</p>
                      <button 
                        className="btn btn-primary" 
                        style={{ marginTop: "16px" }}
                        onClick={() => setShowSeedModal(true)}
                      >
                        Populate Data
                      </button>
                    </div>
                  ) : (
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>Order #</th>
                          <th>Amount</th>
                          <th>Currency</th>
                          <th>Status</th>
                          <th>Closed</th>
                          <th>Created</th>
                        </tr>
                      </thead>
                      <tbody>
                        {orders.map((o) => (
                          <tr key={o.id}>
                            <td style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: "12px" }}>{o.orderNumber}</td>
                            <td style={{ fontWeight: 500, color: "var(--ink)" }}>{fmt.currency(o.amountCents, o.currency)}</td>
                            <td>{o.currency}</td>
                            <td><StatusBadge s={o.status} /></td>
                            <td style={{ color: "var(--mute)" }}>{fmt.date(o.closedAt ?? "")}</td>
                            <td style={{ color: "var(--mute)" }}>{fmt.date(o.createdAt)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              )}
            </>
          )}
        </main>
      </div>

      {/* Seed Modal */}
      {showSeedModal && (
        <div style={{ position: "fixed", top: 0, left: 0, right: 0, bottom: 0, background: "rgba(0,0,0,0.6)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 9999 }}>
          <div style={{ background: "var(--canvas)", padding: "var(--space-xl)", borderRadius: "var(--r-lg)", maxWidth: "500px", width: "100%", border: "1px solid var(--hairline)", boxShadow: "var(--shadow-5)" }}>
            <h2 className="display-sm" style={{ display: "flex", alignItems: "center", gap: "8px", margin: "0 0 16px 0", color: "var(--ink)" }}>
              <span style={{ color: "var(--success)" }}>●</span> 
              {isSeeding ? "Seeding in Progress..." : "Recommended: Populate Synthetic Data"}
            </h2>
            
            {!isSeeding ? (
              <>
                <p style={{ color: "var(--mute)", lineHeight: 1.5, marginBottom: "24px" }}>
                  Are you sure you want to populate synthetic data (Orders, MRR, Tickets) for all Salesforce accounts? 
                  <br/><br/>
                  <strong style={{ color: "var(--error)" }}>WARNING:</strong> This will wipe out any existing orders and Jira tickets across your project to provide a fresh, heavily-populated dataset for the MCP Agent to analyze.
                </p>
                <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
                  <button className="btn btn-secondary" onClick={() => setShowSeedModal(false)}>
                    Cancel
                  </button>
                  <button className="btn btn-primary" onClick={handleSeedData}>
                    Yes, Populate Orders
                  </button>
                </div>
              </>
            ) : (
              <div style={{ marginTop: "24px" }}>
                <div style={{ width: "100%", height: "8px", background: "var(--canvas-soft-2)", borderRadius: "var(--r-full)", overflow: "hidden", border: "1px solid var(--hairline)" }}>
                  <div style={{ 
                    width: seedProgress && seedProgress.total > 0 ? `${(seedProgress.completed / seedProgress.total) * 100}%` : "0%", 
                    height: "100%", 
                    background: "var(--link)", 
                    transition: "width 0.3s ease-out" 
                  }} />
                </div>
                <p style={{ color: "var(--mute)", fontSize: "13px", marginTop: "12px", textAlign: "right", fontFamily: "'JetBrains Mono', monospace" }}>
                  {seedProgress ? `${seedProgress.completed} / ${seedProgress.total} Accounts` : "Initializing..."}
                </p>
                <div style={{
                  marginTop: "16px",
                  background: "#0d1117",
                  border: "1px solid var(--hairline-strong)",
                  borderRadius: "var(--r-md)",
                  padding: "12px",
                  height: "200px",
                  overflowY: "auto",
                  fontFamily: "'JetBrains Mono', monospace",
                  fontSize: "12px",
                  color: "#e6edf3",
                  display: "flex",
                  flexDirection: "column",
                  gap: "4px"
                }}>
                  {seedProgress?.logs && seedProgress.logs.length > 0 ? (
                    seedProgress.logs.map((log, idx) => (
                      <div key={idx} style={{ 
                        color: log.startsWith("FAIL:") ? "#ff7b72" : log.startsWith("SUCCESS:") ? "#7ee787" : "#e6edf3",
                        opacity: log.startsWith("START:") ? 0.7 : 1
                      }}>
                        {log}
                      </div>
                    ))
                  ) : (
                    <div style={{ color: "var(--mute)" }}>Waiting for server...</div>
                  )}
                  <div ref={logsEndRef} />
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
