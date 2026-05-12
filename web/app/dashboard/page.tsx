"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getUsage, getTenant } from "@/lib/api";
import { BarChart3, Activity, Radio, Wifi, ArrowUp, LayoutDashboard, Key, LogOut } from "lucide-react";
import Link from "next/link";
import { toast } from "sonner";

interface Stats { total_connects: number; total_disconnects: number; total_subscribes: number; total_publishes: number; payload_bytes: number; }

export default function DashboardPage() {
  const [tenant, setTenant] = useState<{ name: string; plan: string; api_key: string } | null>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [period, setPeriod] = useState("24h");
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const apiKey = typeof window !== "undefined" ? sessionStorage.getItem("tenant_api_key") : null;
  const tenantId = typeof window !== "undefined" ? sessionStorage.getItem("tenant_id") : null;

  useEffect(() => {
    if (!apiKey || !tenantId) { router.push("/login"); return; }
    Promise.all([getTenant(apiKey), getUsage(tenantId, period)])
      .then(([td, ud]) => {
        setTenant(td.tenant);
        setStats(ud.usage);
      })
      .catch(() => toast.error("Failed to load dashboard — is the backend running?"))
      .finally(() => setLoading(false));
  }, [apiKey, tenantId, period, router]);

  const logout = () => {
    sessionStorage.clear();
    router.push("/login");
  };

  if (loading) return <div className="flex min-h-[60vh] items-center justify-center text-[13px] text-[var(--muted)]">Loading dashboard...</div>;

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      {/* Top bar */}
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{tenant?.name || "Dashboard"}</h1>
          <p className="text-[12px] text-[var(--muted)]">{tenant?.plan} plan</p>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/settings" className="flex items-center gap-1.5 text-[12px] text-[var(--muted)] transition-colors hover:text-[var(--foreground)]">
            <Key className="h-3.5 w-3.5" /> Settings
          </Link>
          <button onClick={logout} className="flex items-center gap-1.5 text-[12px] text-[var(--muted)] transition-colors hover:text-[var(--error)]">
            <LogOut className="h-3.5 w-3.5" /> Logout
          </button>
        </div>
      </div>

      {/* Stats grid */}
      <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "Connections", val: (stats?.total_connects ?? 0) + (stats?.total_disconnects ?? 0), icon: Wifi, color: "text-blue-500" },
          { label: "Subscribes", val: stats?.total_subscribes ?? 0, icon: Radio, color: "text-purple-500" },
          { label: "Publishes", val: stats?.total_publishes ?? 0, icon: ArrowUp, color: "text-emerald-500" },
          { label: "Payload", val: stats?.payload_bytes ?? 0, fmt: formatBytes, icon: BarChart3, color: "text-orange-500" },
        ].map((s) => (
          <div key={s.label} className="rounded border border-[var(--border)] bg-[var(--surface)] p-4">
            <div className="flex items-center justify-between">
              <span className="text-[11px] uppercase tracking-[0.05em] text-[var(--muted)]">{s.label}</span>
              <s.icon className={`h-4 w-4 ${s.color}`} />
            </div>
            <p className="mt-2 text-2xl font-semibold">
              {s.fmt ? s.fmt(s.val as number) : (s.val as number).toLocaleString()}
            </p>
          </div>
        ))}
      </div>

      {/* Period selector */}
      <div className="mb-6">
        <h2 className="mb-3 text-[14px] font-semibold">Usage Period</h2>
        <div className="flex gap-2">
          {["1h", "24h", "168h", "720h"].map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`rounded px-3 py-1.5 text-[12px] font-medium transition-colors ${period === p ? "bg-[var(--accent)] text-white" : "border border-[var(--border)] text-[var(--muted)] hover:text-[var(--foreground)]"}`}
            >
              {p === "1h" ? "1 Hour" : p === "24h" ? "24 Hours" : p === "168h" ? "7 Days" : "30 Days"}
            </button>
          ))}
        </div>
      </div>

      {/* Activity log placeholder */}
      <div className="rounded border border-[var(--border)] bg-[var(--surface)] p-6">
        <h2 className="mb-3 text-[14px] font-semibold">Quick Start</h2>
        <div className="rounded border border-[var(--border)] bg-[var(--background)] p-3">
          <code className="text-[12px] text-[var(--muted)] break-all">
            # Connect via WebSocket with your tenant API key<br/>
            ws://localhost:4001/ws?token=YOUR_JWT
          </code>
        </div>
        <p className="mt-3 text-[12px] text-[var(--muted)]">
          Use the <strong>X-Tenant-Key</strong> header for REST API calls and WebSocket upgrades. Your events will appear on this dashboard.
        </p>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1048576).toFixed(1)} MB`;
}
