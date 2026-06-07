import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  adminLogin,
  adminLogout,
  fetchStatus,
  fetchMetrics,
  fetchPrinters,
  setActivePrinter,
  rotateKey,
  getKeyDownloadUrl,
  getKeyQRUrl,
  type AdminStatus,
  type SpoolerMetrics,
  type PrinterListResponse,
} from "@/lib/admin-api";

function formatUptime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

function formatElapsed(ms: number): string {
  if (ms <= 0) return "Never";
  const totalSeconds = Math.floor(ms / 1000);
  const h = Math.floor(totalSeconds / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  const s = totalSeconds % 60;
  if (h > 0) return `${h}h ${m}m ${s}s ago`;
  if (m > 0) return `${m}m ${s}s ago`;
  if (s > 0) return `${s}s ago`;
  return "just now";
}

export default function Admin() {
  const [authenticated, setAuthenticated] = useState(false);
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState<AdminStatus | null>(null);
  const [metrics, setMetrics] = useState<SpoolerMetrics | null>(null);
  const [printers, setPrinters] = useState<PrinterListResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [sessionChecking, setSessionChecking] = useState(true);
  const [qrModal, setQrModal] = useState<{ url: string; label: string } | null>(null);

  const handleLogin = async () => {
    setError("");
    setLoading(true);
    try {
      await adminLogin(token);
      setAuthenticated(true);
    } catch (e) {
      if (e instanceof Error && e.message === "ADMIN_NOT_CONFIGURED") {
        setError(
          "Admin dashboard is not configured on the server. " +
          "Set ADMIN_TOKEN in .env to a secure random string (openssl rand -hex 32) " +
          "and restart the server."
        );
      } else {
        setError("Invalid token. Check your ADMIN_TOKEN environment variable.");
      }
    } finally {
      setLoading(false);
    }
  };

  const refreshData = useCallback(async () => {
    try {
      const [s, m, p] = await Promise.all([
        fetchStatus(),
        fetchMetrics(),
        fetchPrinters(),
      ]);
      setStatus(s);
      setMetrics(m);
      setPrinters(p);
    } catch (e) {
      if (e instanceof Error && e.message === "ADMIN_NOT_CONFIGURED") {
        setAuthenticated(false);
        setError(
          "Admin dashboard is not configured on the server. " +
          "Set ADMIN_TOKEN in .env and restart."
        );
      } else {
        setAuthenticated(false);
        setError("Session expired. Please log in again.");
      }
    }
  }, []);

  // On mount, check if an existing session cookie is still valid
  // (the HttpOnly cookie persists across reloads — server falls back to it)
  useEffect(() => {
    (async () => {
      try {
        const s = await fetchStatus();
        const [m, p] = await Promise.all([fetchMetrics(), fetchPrinters()]);
        setStatus(s);
        setMetrics(m);
        setPrinters(p);
        setAuthenticated(true);
      } catch (e) {
        if (e instanceof Error && e.message === "ADMIN_NOT_CONFIGURED") {
          setError(
            "Admin dashboard is not configured on the server. " +
            "Set ADMIN_TOKEN in .env to a secure random string (openssl rand -hex 32) " +
            "and restart the server."
          );
        }
        // "Unauthorized" or any other error → no valid session, stay on login
      } finally {
        setSessionChecking(false);
      }
    })();
  }, []);

  useEffect(() => {
    if (!authenticated) return;
    refreshData();
    const interval = setInterval(refreshData, 5000);
    return () => clearInterval(interval);
  }, [authenticated, refreshData]);

  const handleSwitchPrinter = async (id: string) => {
    setLoading(true);
    try {
      await setActivePrinter(id);
      await refreshData();
    } catch {
      setError("Failed to switch printer.");
    } finally {
      setLoading(false);
    }
  };

  const handleRotateKey = async () => {
    if (!confirm("Are you sure? This deletes the current key pair. You must restart the server after rotation.")) return;
    setLoading(true);
    try {
      const result = await rotateKey();
      alert(result.message);
    } catch {
      setError("Failed to rotate key.");
    } finally {
      setLoading(false);
    }
  };

  if (!authenticated) {
    // Show a brief loading state while checking for an existing session cookie
    if (sessionChecking) {
      return (
        <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 flex items-center justify-center">
          <Card className="w-full max-w-md">
            <CardContent className="py-8">
              <p className="text-center text-muted-foreground">Checking session...</p>
            </CardContent>
          </Card>
        </div>
      );
    }

    return (
      <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
              ZeroDrop Admin
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="token">Admin Token</Label>
              <Input
                id="token"
                type="password"
                placeholder="Enter Token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleLogin()}
                autoFocus
              />
            </div>
            <Button onClick={handleLogin} disabled={loading || !token} className="w-full">
              Login
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-100">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold">ZeroDrop Admin Dashboard</h1>
            <p className="text-muted-foreground text-sm">
              {status ? `v${status.version} · Uptime: ${formatUptime(status.uptime_seconds)}` : "Loading..."}
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={async () => {
              await adminLogout();
              setToken("");
              setAuthenticated(false);
            }}
          >
            Logout
          </Button>
        </div>

        {error && (
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Monitoring Panel */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Spooler Metrics</CardTitle>
              <CardDescription className="text-xs">
                Real-time print queue stats. Queue depth shows pending jobs out of max capacity.
                Use this to monitor printer health and job throughput.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {metrics ? (
                <>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Queue Depth</span>
                    <span className="font-mono">{metrics.queue_depth} / {metrics.max_queue_depth}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Processed</span>
                    <span className="font-mono text-green-600">{metrics.total_processed}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Total Failed</span>
                    <span className="font-mono text-red-600">{metrics.total_failed}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Last Print</span>
                    <span className="font-mono text-xs">
                      {formatElapsed(metrics.last_print_ms)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Started</span>
                    <span className="font-mono text-xs">{new Date(metrics.spooler_start_time).toLocaleTimeString()}</span>
                  </div>
                </>
              ) : (
                <span className="text-muted-foreground">Loading...</span>
              )}
            </CardContent>
          </Card>

          {/* Printer Panel */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Printers</CardTitle>
              <CardDescription className="text-xs">
                Detected thermal printers. The active printer receives all print jobs.
                Click <strong>Select</strong> to switch between Mock Printer (stdout logging) and a connected USB printer.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {printers ? (
                printers.printers.map((p) => (
                  <div
                    key={p.id}
                    className={`flex items-center justify-between p-2 rounded-md text-sm ${
                      p.id === printers.active_printer
                        ? "bg-blue-50 border border-blue-200"
                        : "hover:bg-gray-50"
                    }`}
                  >
                    <div>
                      <div className="font-medium">{p.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {p.type}{p.device ? ` · ${p.device}` : ""}
                      </div>
                    </div>
                    {p.id !== printers.active_printer && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleSwitchPrinter(p.id)}
                        disabled={loading}
                      >
                        Select
                      </Button>
                    )}
                    {p.id === printers.active_printer && (
                      <span className="text-xs text-blue-600 font-medium">Active</span>
                    )}
                  </div>
                ))
              ) : (
                <span className="text-muted-foreground text-sm">Loading...</span>
              )}
            </CardContent>
          </Card>

          {/* Key Panel */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Key Management</CardTitle>
              <CardDescription className="text-xs">
                Server encryption keys. The public key is shared with submitters via <code>/key</code>. The private key stays on disk — download or scan the QR code for offline decryption in reader.html.
                <strong> Rotate</strong> deletes keys and requires a restart.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {status ? (
                <>
                  <div>
                    <div className="text-muted-foreground mb-1">Fingerprint</div>
                    <code className="text-xs bg-muted px-2 py-1 rounded break-all block">
                      {status.key.fingerprint || "Unavailable"}
                    </code>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Generated</span>
                    <span className="font-mono text-xs">{new Date(status.key.generated_at).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Private Key on Disk</span>
                    <span className={status.key.private_key_on_disk ? "text-green-600" : "text-red-600"}>
                      {status.key.private_key_on_disk ? "Yes" : "No"}
                    </span>
                  </div>
                  <div className="space-y-2 pt-2">
                    {status.key.private_key_on_disk && (
                      <>
                        <Button
                          variant="outline"
                          size="sm"
                          className="w-full"
                          onClick={() => window.open(getKeyDownloadUrl(), "_blank")}
                        >
                          Download Private Key (PEM)
                        </Button>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            className="flex-1"
                            onClick={() => setQrModal({ url: getKeyQRUrl("private_key_jwk_qr.png"), label: "Private Key (JWK)" })}
                          >
                            QR (JWK)
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            className="flex-1"
                            onClick={() => setQrModal({ url: getKeyQRUrl("private_key_qr.png"), label: "Private Key (PEM)" })}
                          >
                            QR (PEM)
                          </Button>
                        </div>
                      </>
                    )}
                    <Button
                      variant="destructive"
                      size="sm"
                      className="w-full"
                      onClick={handleRotateKey}
                      disabled={loading}
                    >
                      Rotate Key
                    </Button>
                  </div>
                </>
              ) : (
                <span className="text-muted-foreground">Loading...</span>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* QR Code Modal Overlay */}
      {qrModal && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
          onClick={() => setQrModal(null)}
        >
          <div
            className="bg-white rounded-xl p-6 max-w-sm mx-4 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold">{qrModal.label}</h3>
              <button
                className="text-gray-400 hover:text-gray-600 text-lg leading-none"
                onClick={() => setQrModal(null)}
              >
                &times;
              </button>
            </div>
            <img
              src={qrModal.url}
              alt={qrModal.label}
              className="w-full h-auto rounded"
            />
          </div>
        </div>
      )}
    </div>
  );
}
