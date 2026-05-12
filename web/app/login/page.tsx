"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getTenant } from "@/lib/api";
import { toast } from "sonner";
import { Zap } from "lucide-react";
import Link from "next/link";

export default function LoginPage() {
  const [apiKey, setApiKey] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiKey.trim()) return;
    setLoading(true);
    try {
      const data = await getTenant(apiKey);
      sessionStorage.setItem("tenant_api_key", apiKey);
      sessionStorage.setItem("tenant_id", data.tenant.id);
      sessionStorage.setItem("tenant_plan", data.tenant.plan);
      toast.success(`Welcome back, ${data.tenant.name}`);
      router.push("/dashboard");
    } catch {
      toast.error("Invalid API key");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-[80vh] items-center justify-center px-4">
      <div className="w-full max-w-[380px]">
        <div className="mb-8 text-center">
          <Zap className="mx-auto mb-3 h-8 w-8 text-[var(--accent)]" />
          <h1 className="text-xl font-semibold">Sign in to your tenant</h1>
          <p className="mt-1.5 text-[13px] text-[var(--muted)]">Enter your API key to access the dashboard.</p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <label className="block space-y-1.5">
            <span className="text-[12px] font-medium">API Key</span>
            <input
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              required
              placeholder="pk_..."
              className="h-9 w-full rounded border border-[var(--border)] bg-[var(--surface)] px-3 text-[13px] font-mono outline-none focus:border-[var(--accent)]"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="flex h-9 w-full items-center justify-center rounded bg-[var(--accent)] text-[14px] font-medium text-white transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-60"
          >
            {loading ? "Verifying..." : "Access Dashboard"}
          </button>
        </form>
        <p className="mt-4 text-center text-[12px] text-[var(--muted)]">
          Don&apos;t have a tenant? <Link href="/signup" className="text-[var(--accent)] hover:underline">Create one</Link>
        </p>
      </div>
    </div>
  );
}
