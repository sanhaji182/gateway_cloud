"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { registerTenant } from "@/lib/api";
import { toast } from "sonner";
import { Zap, Loader2 } from "lucide-react";
import Link from "next/link";

export default function SignupPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !email.trim()) return;
    setLoading(true);
    try {
      const data = await registerTenant(name, email);
      toast.success("Tenant registered! Save your API key.");
      sessionStorage.setItem("tenant_api_key", data.api_key);
      sessionStorage.setItem("tenant_id", data.tenant.id);
      router.push(`/dashboard`);
    } catch (err) {
      toast.error("Registration failed. Is the backend running?");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-[80vh] items-center justify-center px-4">
      <div className="w-full max-w-[380px]">
        <div className="mb-8 text-center">
          <Zap className="mx-auto mb-3 h-8 w-8 text-[var(--accent)]" />
          <h1 className="text-xl font-semibold">Create your free tenant</h1>
          <p className="mt-1.5 text-[13px] text-[var(--muted)]">Get started in seconds. No credit card required.</p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <label className="block space-y-1.5">
            <span className="text-[12px] font-medium">Project Name</span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              placeholder="My SaaS App"
              className="h-9 w-full rounded border border-[var(--border)] bg-[var(--surface)] px-3 text-[13px] outline-none focus:border-[var(--accent)]"
            />
          </label>
          <label className="block space-y-1.5">
            <span className="text-[12px] font-medium">Email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="you@example.com"
              className="h-9 w-full rounded border border-[var(--border)] bg-[var(--surface)] px-3 text-[13px] outline-none focus:border-[var(--accent)]"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="flex h-9 w-full items-center justify-center rounded bg-[var(--accent)] text-[14px] font-medium text-white transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-60"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Create Tenant"}
          </button>
        </form>
        <p className="mt-4 text-center text-[12px] text-[var(--muted)]">
          Already have an API key? <Link href="/login" className="text-[var(--accent)] hover:underline">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
