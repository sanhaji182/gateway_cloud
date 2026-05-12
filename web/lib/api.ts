const API = process.env.NEXT_PUBLIC_CLOUD_API || "http://localhost:4001";

export async function registerTenant(name: string, email: string) {
  const res = await fetch(`${API}/api/cloud/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, email }),
  });
  if (!res.ok) throw new Error("Registration failed");
  return res.json();
}

export async function getTenant(apiKey: string) {
  const res = await fetch(`${API}/api/cloud/tenant?api_key=${encodeURIComponent(apiKey)}`);
  if (!res.ok) throw new Error("Tenant not found");
  return res.json();
}

export async function getUsage(tenantId: string, period = "24h") {
  const res = await fetch(`${API}/api/cloud/usage?tenant_id=${tenantId}&period=${period}`);
  if (!res.ok) throw new Error("Usage fetch failed");
  return res.json();
}
