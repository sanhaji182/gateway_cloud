import Link from "next/link";
import { Zap, Shield, BarChart3, Globe, ArrowRight, Check } from "lucide-react";

const plans = [
  {
    name: "Free", price: "$0", desc: "For side projects and prototyping",
    events: "100", connections: "5",
    features: ["1 tenant", "WebSocket gateway", "Usage analytics", "Community support"],
  },
  {
    name: "Pro", price: "$29", desc: "For growing applications",
    events: "10,000", connections: "1,000",
    features: ["Unlimited tenants", "Priority support", "99.9% SLA", "Stripe billing", "Webhook delivery"],
  },
  {
    name: "Enterprise", price: "Custom", desc: "For high-scale production",
    events: "100,000", connections: "10,000",
    features: ["Everything in Pro", "Dedicated support", "Custom SLA", "On-prem deployment", "SSO / SAML"],
  },
];

const features = [
  { icon: Zap, title: "Realtime WebSocket", desc: "Low-latency pub/sub with Redis. Public, private, presence, and wildcard channels." },
  { icon: Shield, title: "Multi-Tenant Auth", desc: "Per-tenant API keys with X-Tenant-Key header. Isolated rate limits and usage tracking." },
  { icon: BarChart3, title: "Usage Analytics", desc: "Real-time dashboard with events/minute, connection counts, and payload bytes per tenant." },
  { icon: Globe, title: "Self-Hosted or Cloud", desc: "Run on your own infra or use our managed cloud. Same API, zero lock-in." },
];

export default function Home() {
  return (
    <div className="mx-auto max-w-6xl px-4">
      {/* Hero */}
      <section className="flex flex-col items-center py-20 text-center">
        <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-5xl">
          <span className="gradient-hero">Realtime Infrastructure</span>
          <br />for your next big idea
        </h1>
        <p className="mt-4 max-w-xl text-[15px] text-[var(--muted)]">
          Add realtime features to your app in minutes — WebSocket pub/sub, presence channels,
          webhooks, and usage analytics. Free tier, no credit card required.
        </p>
        <div className="mt-8 flex gap-3">
          <Link href="/signup" className="inline-flex items-center gap-1.5 rounded bg-[var(--accent)] px-5 py-2.5 text-[14px] font-medium text-white transition-colors hover:bg-[var(--accent-hover)]">
            Get Started Free <ArrowRight className="h-4 w-4" />
          </Link>
          <Link href="/login" className="inline-flex items-center rounded border border-[var(--border)] px-5 py-2.5 text-[14px] font-medium transition-colors hover:bg-[var(--surface)]">
            Sign In
          </Link>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="py-16">
        <h2 className="text-center text-2xl font-semibold">Why Gateway Cloud?</h2>
        <div className="mt-10 grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {features.map((f) => (
            <div key={f.title} className="rounded border border-[var(--border)] bg-[var(--surface)] p-6">
              <f.icon className="mb-3 h-6 w-6 text-[var(--accent)]" />
              <h3 className="text-[14px] font-semibold">{f.title}</h3>
              <p className="mt-1.5 text-[13px] text-[var(--muted)]">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" className="py-16">
        <h2 className="text-center text-2xl font-semibold">Simple, transparent pricing</h2>
        <div className="mt-10 grid gap-6 md:grid-cols-3">
          {plans.map((p) => (
            <div key={p.name} className={`rounded border bg-[var(--surface)] p-6 ${p.name === "Pro" ? "border-[var(--accent)] ring-1 ring-[var(--accent)]" : "border-[var(--border)]"}`}>
              <h3 className="text-[16px] font-semibold">{p.name}</h3>
              <div className="mt-2">
                <span className="text-3xl font-bold">{p.price}</span>
                {p.price !== "Custom" && <span className="text-[13px] text-[var(--muted)]">/month</span>}
              </div>
              <p className="mt-1 text-[12px] text-[var(--muted)]">{p.desc}</p>
              <div className="mt-4 space-y-1.5 border-y border-[var(--border)] py-3 text-[13px]">
                <p><strong>{p.events}</strong> events/min</p>
                <p><strong>{p.connections}</strong> connections</p>
              </div>
              <ul className="mt-4 space-y-2">
                {p.features.map((f) => (
                  <li key={f} className="flex items-center gap-2 text-[13px] text-[var(--muted)]">
                    <Check className="h-3.5 w-3.5 text-[var(--success)]" /> {f}
                  </li>
                ))}
              </ul>
              <Link href="/signup" className="mt-6 flex items-center justify-center rounded border border-[var(--border)] py-2 text-[13px] font-medium transition-colors hover:bg-[var(--accent)] hover:text-white hover:border-[var(--accent)]">
                Get Started
              </Link>
            </div>
          ))}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-[var(--border)] py-8 text-center text-[12px] text-[var(--muted)]">
        <p>
          Built by{" "}
          <a href="https://www.linkedin.com/in/sansanhaji/" className="underline hover:text-[var(--foreground)]">Sonick Sanhaji</a>
          {" — "}AI-assisted, human-architected.
        </p>
      </footer>
    </div>
  );
}
