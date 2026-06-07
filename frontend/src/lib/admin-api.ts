const API_BASE = "/api/admin";

// In-memory session token. Never stored in localStorage/cookies — exists only
// for the lifetime of the page. The server also sets an HttpOnly cookie as
// fallback, but header-based auth is more reliable in the fetch API.
let sessionToken = "";

export interface AdminStatus {
  version: string;
  uptime_seconds: number;
  key: {
    fingerprint: string;
    generated_at: string;
    private_key_on_disk: boolean;
  };
}

export interface SpoolerMetrics {
  queue_depth: number;
  max_queue_depth: number;
  total_processed: number;
  total_failed: number;
  last_print_time: string;
  last_print_ms: number;
  spooler_start_time: string;
}

export interface PrinterInfo {
  id: string;
  name: string;
  type: string;
  device: string;
}

export interface PrinterListResponse {
  printers: PrinterInfo[];
  active_printer: string;
}

/** Build headers with the session token for authenticated requests. */
function authHeaders(): HeadersInit {
  const headers: Record<string, string> = {};
  if (sessionToken) {
    headers["X-Session-Token"] = sessionToken;
  }
  return headers;
}

export async function adminLogin(token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("ADMIN_NOT_CONFIGURED");
    }
    throw new Error("Invalid token");
  }
  // Extract session token from response body
  const body = await res.json();
  sessionToken = body.session || "";
}

export async function fetchStatus(): Promise<AdminStatus> {
  const res = await fetch(`${API_BASE}/status`, { headers: authHeaders() });
  if (!res.ok) {
    if (res.status === 404) throw new Error("ADMIN_NOT_CONFIGURED");
    throw new Error("Unauthorized");
  }
  return res.json();
}

export async function fetchMetrics(): Promise<SpoolerMetrics> {
  const res = await fetch(`${API_BASE}/metrics`, { headers: authHeaders() });
  if (!res.ok) {
    if (res.status === 404) throw new Error("ADMIN_NOT_CONFIGURED");
    throw new Error("Unauthorized");
  }
  return res.json();
}

export async function fetchPrinters(): Promise<PrinterListResponse> {
  const res = await fetch(`${API_BASE}/printers`, { headers: authHeaders() });
  if (!res.ok) {
    if (res.status === 404) throw new Error("ADMIN_NOT_CONFIGURED");
    throw new Error("Unauthorized");
  }
  return res.json();
}

export async function setActivePrinter(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/printers/active`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify({ id }),
  });
  if (!res.ok) throw new Error("Failed to set active printer");
}

export async function rotateKey(): Promise<{ message: string }> {
  const res = await fetch(`${API_BASE}/key/rotate`, {
    method: "POST",
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error("Failed to rotate key");
  return res.json();
}

export function getKeyDownloadUrl(): string {
  return `${API_BASE}/key`;
}

export function getKeyQRUrl(file: string): string {
  return `${API_BASE}/key/qr?file=${file}`;
}
