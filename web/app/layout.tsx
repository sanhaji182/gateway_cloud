import type { Metadata } from "next";
import "./globals.css";
import { Header } from "@/components/Header";
import { Toaster } from "sonner";

export const metadata: Metadata = {
  title: "Gateway Cloud — Realtime SaaS",
  description: "Self-hosted realtime infrastructure. Free tier, Pro & Enterprise plans.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-[var(--background)] text-[var(--foreground)] antialiased">
        <Header />
        <main>{children}</main>
        <Toaster position="top-right" richColors />
      </body>
    </html>
  );
}
