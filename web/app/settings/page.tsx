"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getTenant } from "@/lib/api";
import { Copy, ArrowLeft, Check } from "lucide-react";
import Link from "next/link";
import { toast } from "sonner";

export default function SettingsPage() {
  const [tenant, setTenant] = useState<{ id: string; name: string; plan: string; api_key: string; created_at: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const router = useRouter();
  const apiKey = typeof window !== "undefined" ? sessionStorage.getItem("tenant_api_key") : null;

  useEffect(() => {
    if (!apiKey) { router.push("/login"); return; }
    getTenant(apiKey)
      .then((d) => setTenant(d.tenant))
      .catch(() => toast.error("Failed to load tenant info"));
  }, [apiKey, router]);

  const copyKey = () => {
    if (!tenant?.api_key) return;
    navigator.clipboard.writeText(tenant.api_key);
    setCopied(true);
    toast.success("API key copied");
    setTimeout(() => setCopied(false), 2000);
  };

  const plans = [
    { name: "Free", events: "100/min", conn: "5", price: "$0" },
    { name: "Pro", events: "10k/min", conn: "1,000", price: "$29/mo" },
    { name: "Enterprise", events: "100k/min", conn: "10,000", price: "Custom" },
  ];

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <Link href="/dashboard" className="mb-6 flex items-center gap-1.5 text-[12px] text-[var(--muted)] hover:text-[var(--foreground)]">
        <ArrowLeft className="h-3.5 w-3.5" /> Back to Dashboard
      </Link>

      <h1 className="mb-6 text-xl font-semibold">Tenant Settings</h1>

      {/* Tenant Info */}
      {tenant && (
        <div className="mb-6 rounded border border-[var(--border)] bg-[var(--surface)] p-5">
          <h2 className="mb-3 text-[14px] font-semibold">Tenant Info</h2>
          <div className="space-y-2 text-[13px]">
            <div className="flex justify-between"><span className="text-[var(--muted)]">Name</span><span>{tenant.name}</span></div>
            <div className="flex justify-between"><span className="text-[var(--muted)]">Plan</span><span className="font-medium capitalize">{tenant.plan}</span></div>
            <div className="flex justify-between"><span className="text-[var(--muted)]">Created</span><span>{new Date(tenant.created_at).toLocaleDateString()}</span></div>
          </div>
        </div>
      )}

      {/* API Key */}
      <div className="mb-6 rounded border border-[var(--border)] bg-[var(--surface)] p-5">
        <h2 className="mb-3 text-[14px] font-semibold">API Key</h2>
        <p className="mb-2 text-[12px] text-[var(--muted)]">Use this key in the <code className="rounded bg-[var(--background)] px-1">X-Tenant-Key</code> header for all API requests.</p>
        <div className="flex items-center gap-2">
          <code className="flex-1 overflow-x-auto rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[12px] font-mono whitespace-nowrap select-all">
            {tenant?.api_key}
          </code>
          <button
            onClick={copyKey}
            className="flex h-9 w-9 items-center justify-center rounded border border-[var(--border)] transition-colors hover:bg-[var(--background)]"
            title="Copy API key"
          >
            {copied ? <Check className="h-4 w-4 text-[var(--success)]" /> : <Copy className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {/* Plan Tiers / Upgrade */}
      <div className="rounded border border-[var(--border)] bg-[var(--surface)] p-5">
        <h2 className="mb-3 text-[14px] font-semibold">Plan & Upgrade</h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {plans.map((p) => (
            <div key={p.name} className={`rounded border p-3 text-center ${tenant?.plan === p.name.toLowerCase() ? "border-[var(--accent)] ring-1 ring-[var(--accent)]" : "border-[var(--border)]"}`}>
              <h3 className="text-[13px] font-semibold">{p.name}</h3>
              <p className="mt-0.5 text-lg font-bold">{p.price}</p>
              <p className="text-[11px] text-[var(--muted)]">{p.events} · {p.conn} conn</p>
              {tenant?.plan === p.name.toLowerCase() && <span className="mt-2 inline-block rounded-full bg-[var(--accent)]/10 px-2 py-0.5 text-[10px] font-medium text-[var(--accent)]">Current</span>}
            </div>
          ))}
        </div>
        <p className="mt-4 text-[12px] text-[var(--muted)]">
          {tenant?.plan === "free"
            ? "Upgrade to Pro for unlimited tenants and 10,000 events/min. Stripe integration coming soon."
            : tenant?.plan === "pro"
            ? "Need more scale? Contact us for Enterprise pricing."
            : "You're on the Enterprise plan. Custom SLA and dedicated support."}
        </p>
      </div>
    </div>
  );
}
