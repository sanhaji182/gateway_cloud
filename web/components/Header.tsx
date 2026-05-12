"use client";

import Link from "next/link";
import { Zap, Menu, X } from "lucide-react";
import { useState } from "react";

export function Header() {
  const [open, setOpen] = useState(false);
  const links = [
    { href: "/#pricing", label: "Pricing" },
    { href: "/#features", label: "Features" },
    { href: "/signup", label: "Get Started" },
    { href: "/login", label: "Sign In" },
  ];

  return (
    <header className="sticky top-0 z-50 border-b border-[var(--border)] bg-[var(--surface)]/80 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
        <Link href="/" className="flex items-center gap-2 font-semibold text-[15px]">
          <Zap className="h-5 w-5 text-[var(--accent)]" />
          Gateway Cloud
        </Link>
        <nav className="hidden items-center gap-6 md:flex">
          {links.map((l) => (
            <Link key={l.href} href={l.href} className="text-[13px] text-[var(--muted)] transition-colors hover:text-[var(--foreground)]">
              {l.label}
            </Link>
          ))}
        </nav>
        <button className="md:hidden" onClick={() => setOpen(!open)} aria-label="Toggle menu">
          {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>
      {open && (
        <div className="border-t border-[var(--border)] bg-[var(--surface)] px-4 py-3 md:hidden">
          {links.map((l) => (
            <Link key={l.href} href={l.href} className="block py-2 text-[13px] text-[var(--muted)]" onClick={() => setOpen(false)}>
              {l.label}
            </Link>
          ))}
        </div>
      )}
    </header>
  );
}
