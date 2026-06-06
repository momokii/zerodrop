const API_BASE = "/api/admin";

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

export async function adminLogin(token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  if (!res.ok) throw new Error("Invalid token");
}

export async function fetchStatus(): Promise<AdminStatus> {
  const res = await fetch(`${API_BASE}/status`);
  if (!res.ok) throw new Error("Unauthorized");
  return res.json();
}

export async function fetchMetrics(): Promise<SpoolerMetrics> {
  const res = await fetch(`${API_BASE}/metrics`);
  if (!res.ok) throw new Error("Unauthorized");
  return res.json();
}

export async function fetchPrinters(): Promise<PrinterListResponse> {
  const res = await fetch(`${API_BASE}/printers`);
  if (!res.ok) throw new Error("Unauthorized");
  return res.json();
}

export async function setActivePrinter(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/printers/active`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
  if (!res.ok) throw new Error("Failed to set active printer");
}

export async function rotateKey(): Promise<{ message: string }> {
  const res = await fetch(`${API_BASE}/key/rotate`, { method: "POST" });
  if (!res.ok) throw new Error("Failed to rotate key");
  return res.json();
}

export function getKeyDownloadUrl(): string {
  return `${API_BASE}/key`;
}

export function getKeyQRUrl(file: string): string {
  return `${API_BASE}/key/qr?file=${file}`;
}
