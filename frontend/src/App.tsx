import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { checkHealth, fetchPublicKey, submitPayload, validatePayload } from "@/lib/api";
import { encryptData, calculateFingerprintFromPEM, parsePEM } from "@/lib/crypto";
import { copyToClipboard } from "@/lib/utils";

type Step = "encrypt" | "submit" | "success" | "error";

interface Status {
  type: "info" | "success" | "error" | "warning";
  title: string;
  message: string;
}

function App() {
  const [step, setStep] = useState<Step>("encrypt");
  const [plaintext, setPlaintext] = useState("");
  const [encryptedData, setEncryptedData] = useState("");
  const [serverPublicKey, setServerPublicKey] = useState("");
  const [keyFingerprint, setKeyFingerprint] = useState("");
  const [status, setStatus] = useState<Status | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isHealthy, setIsHealthy] = useState(false);
  const [printerInfo, setPrinterInfo] = useState<string>("");

  // Fetch server public key and health on mount
  useEffect(() => {
    async function init() {
      try {
        // Web Crypto API (crypto.subtle) requires a secure context:
        // HTTPS or http://localhost. Accessing from other devices or
        // over plain HTTP makes crypto.subtle undefined.
        if (!window.crypto.subtle) {
          setStatus({
            type: "error",
            title: "Secure Context Required",
            message: "This application requires HTTPS or localhost access. Web Crypto API is not available over plain HTTP from remote devices.",
          });
          return;
        }

        // Check health first
        const health = await checkHealth();
        setIsHealthy(true);
        if (health.printer) {
          setPrinterInfo(`${health.printer.type} printer (${health.printer.available ? "available" : "unavailable"})`);
        }

        // Fetch public key
        const publicKey = await fetchPublicKey();
        setServerPublicKey(publicKey);

        // Calculate fingerprint
        const fingerprint = await calculateFingerprintFromPEM(publicKey);
        setKeyFingerprint(fingerprint);

        setStatus({
          type: "info",
          title: "Connected to ZeroDrop Terminal",
          message: "Server is ready. Enter your message below to encrypt and submit.",
        });
      } catch (error) {
        setStatus({
          type: "error",
          title: "Connection Error",
          message: error instanceof Error ? error.message : "Failed to connect to server",
        });
      }
    };

    init();
  }, []);

  const handleEncrypt = async () => {
    if (!plaintext.trim()) {
      setStatus({
        type: "error",
        title: "Input Required",
        message: "Please enter a message to encrypt.",
      });
      return;
    }

    try {
      setStatus({
        type: "info",
        title: "Encrypting...",
        message: "Encrypting...",
      });

      // Parse PEM and import the server's public key
      const derBytes = parsePEM(serverPublicKey);
      const publicKey = await crypto.subtle.importKey(
        "spki",
        derBytes,
        { name: "X25519" },
        false,
        []
      );

      // Encrypt the data
      const result = await encryptData(plaintext, publicKey);
      setEncryptedData(result.qrData);
      setStep("submit");

      setStatus({
        type: "success",
        title: "Encryption Complete",
        message: `Your message has been encrypted. Review the encrypted data below, then submit to print.`,
      });
    } catch (error) {
      setStatus({
        type: "error",
        title: "Encryption Failed",
        message: error instanceof Error ? error.message : "Failed to encrypt message",
      });
    }
  };

  const handleSubmit = async () => {
    const validation = validatePayload(encryptedData);
    if (!validation.valid) {
      setStatus({
        type: "error",
        title: "Invalid Payload",
        message: validation.error || "Payload validation failed",
      });
      return;
    }

    setIsSubmitting(true);
    try {
      setStatus({
        type: "info",
        title: "Transmitting...",
        message: "Transmitting...",
      });

      await submitPayload(encryptedData);
      setStep("success");

      setStatus({
        type: "success",
        title: "Safely Dropped.",
        message: "Safely Dropped.",
      });
    } catch (error) {
      setStep("submit");
      setStatus({
        type: "error",
        title: "Submission Failed",
        message: error instanceof Error ? error.message : "Failed to submit payload",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCopyKey = async () => {
    const success = await copyToClipboard(serverPublicKey);
    if (success) {
      setStatus({
        type: "success",
        title: "Copied!",
        message: "Public key copied to clipboard.",
      });
    }
  };

  const handleReset = () => {
    setPlaintext("");
    setEncryptedData("");
    setStep("encrypt");
    setStatus({
      type: "info",
      title: "Ready",
      message: "Enter a new message to encrypt and submit.",
    });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <div className="container mx-auto px-4 py-8 max-w-2xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center mb-4">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-600 to-blue-600 rounded-xl flex items-center justify-center shadow-lg">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
          </div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent">
            ZeroDrop Terminal
          </h1>
          <p className="text-muted-foreground mt-2">
            Zero-Knowledge Secure Credential Delivery
          </p>
          {printerInfo && (
            <div className="inline-flex items-center gap-2 mt-2 px-3 py-1 bg-green-100 text-green-700 rounded-full text-sm">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              {printerInfo}
            </div>
          )}
        </div>

        {/* Status Alert */}
        {status && (
          <Alert variant={status.type === "error" ? "destructive" : "default"} className="mb-6">
            <AlertTitle>{status.title}</AlertTitle>
            <AlertDescription>{status.message}</AlertDescription>
          </Alert>
        )}

        {/* Key Fingerprint */}
        {keyFingerprint && (
          <Card className="mb-6 border-purple-200 dark:border-purple-800">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center justify-between">
                <span>Server Public Key Fingerprint</span>
                <Button variant="ghost" size="sm" onClick={handleCopyKey}>
                  <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  Copy
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <code className="text-xs bg-muted px-2 py-1 rounded break-all">
                SHA-256:{keyFingerprint}
              </code>
            </CardContent>
          </Card>
        )}

        {/* Main Card */}
        <Card className="shadow-lg">
          <CardHeader>
            <CardTitle>
              {step === "encrypt" && "Encrypt Your Message"}
              {step === "submit" && "Review & Submit"}
              {step === "success" && "Print Queued Successfully"}
              {step === "error" && "Error Occurred"}
            </CardTitle>
            <CardDescription>
              {step === "encrypt" &&
                "Enter your sensitive message below. It will be encrypted in your browser using the server's public key."}
              {step === "submit" &&
                "Review the encrypted data below. Once submitted, it will be printed as a QR code."}
              {step === "success" &&
                "Your encrypted message has been added to the print queue."}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {step === "encrypt" && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="message">Your Message</Label>
                  <Textarea
                    id="message"
                    placeholder="Enter your sensitive message (password, API key, security report, etc.)..."
                    value={plaintext}
                    onChange={(e) => setPlaintext(e.target.value)}
                    rows={6}
                    className="resize-none"
                  />
                  <p className="text-xs text-muted-foreground">
                    {plaintext.length} characters
                  </p>
                </div>

                <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-blue-600 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <div className="text-sm">
                      <p className="font-medium text-blue-900 dark:text-blue-100">Zero-Knowledge Guarantee</p>
                      <p className="text-blue-700 dark:text-blue-300 mt-1">
                        Your message is encrypted in your browser using Web Crypto API. The server never sees your plaintext message and cannot decrypt it.
                      </p>
                    </div>
                  </div>
                </div>

                <Button onClick={handleEncrypt} className="w-full" size="lg">
                  <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                  Encrypt Message
                </Button>
              </>
            )}

            {step === "submit" && (
              <>
                <div className="space-y-2">
                  <Label>Encrypted Data</Label>
                  <div className="bg-muted rounded-lg p-4">
                    <code className="text-xs break-all block">
                      {encryptedData}
                    </code>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {encryptedData.length} characters (QR version header included)
                  </p>
                </div>

                <div className="flex gap-3">
                  <Button
                    onClick={() => setStep("encrypt")}
                    variant="outline"
                    className="flex-1"
                  >
                    Back
                  </Button>
                  <Button
                    onClick={handleSubmit}
                    disabled={isSubmitting}
                    className="flex-1"
                  >
                    {isSubmitting ? (
                      <>
                        <svg className="w-4 h-4 mr-2 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Submitting...
                      </>
                    ) : (
                      <>
                        <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Submit to Print
                      </>
                    )}
                  </Button>
                </div>
              </>
            )}

            {step === "success" && (
              <>
                <div className="text-center py-6">
                  <div className="w-16 h-16 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center mx-auto mb-4">
                    <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <h3 className="text-lg font-semibold mb-2">Print Job Queued</h3>
                  <p className="text-muted-foreground text-sm">
                    Your encrypted message will be printed as a QR code shortly.
                  </p>
                </div>

                <Button onClick={handleReset} className="w-full">
                  Submit Another Message
                </Button>
              </>
            )}
          </CardContent>
        </Card>

        {/* Footer */}
        <div className="mt-8 text-center text-sm text-muted-foreground">
          <p className="mb-2">
            <strong>ZeroDrop Terminal v1.0</strong> — Zero-Knowledge Secure Credential Delivery
          </p>
          <p>
            All encryption happens in your browser. Server cannot decrypt your messages.
          </p>
        </div>
      </div>
    </div>
  );
}

export default App;
