/**
 * ZeroDrop API Service
 * Handles communication with the ZeroDrop backend
 */

const API_BASE = "";

export interface HealthStatus {
  status: string;
  service: string;
  printer?: {
    type: string;
    available: boolean;
    status: string;
    device_path?: string;
    model?: string;
  };
}

export interface DropResponse {
  status: string;
  message: string;
}

export interface ApiError {
  message: string;
  status?: number;
}

/**
 * Fetch the server's public key
 */
export async function fetchPublicKey(): Promise<string> {
  const response = await fetch(`${API_BASE}/key`);

  if (!response.ok) {
    throw new Error(`Failed to fetch public key: ${response.statusText}`);
  }

  const publicKeyPEM = await response.text();
  return publicKeyPEM.trim();
}

/**
 * Submit an encrypted payload to the server
 */
export async function submitPayload(qrData: string): Promise<DropResponse> {
  const response = await fetch(`${API_BASE}/drop`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      payload: qrData,
    }),
  });

  if (!response.ok) {
    let errorMessage = `Failed to submit payload: ${response.statusText}`;
    try {
      const errorData = await response.json();
      if (errorData.error) {
        errorMessage = errorData.error;
      }
    } catch {
      // Use default error message
    }
    throw new Error(errorMessage);
  }

  return await response.json();
}

/**
 * Check server health
 */
export async function checkHealth(): Promise<HealthStatus> {
  const response = await fetch(`${API_BASE}/health`);

  if (!response.ok) {
    throw new Error(`Health check failed: ${response.statusText}`);
  }

  return await response.json();
}

/**
 * Validate that the payload is properly formatted
 */
export function validatePayload(payload: string): { valid: boolean; error?: string } {
  if (!payload || payload.trim().length === 0) {
    return { valid: false, error: "Payload cannot be empty" };
  }

  if (payload.length > 400) {
    return { valid: false, error: "Payload exceeds 400 character limit" };
  }

  // Check for ZD1: prefix
  if (!payload.startsWith("ZD1:")) {
    return { valid: false, error: 'Payload must start with "ZD1:"' };
  }

  // Validate base64 after prefix
  const base64Part = payload.slice(4);
  try {
    atob(base64Part);
  } catch {
    return { valid: false, error: "Payload contains invalid base64 encoding" };
  }

  return { valid: true };
}

/**
 * Estimate QR code size based on payload length
 * Returns approximate QR code version (1-40)
 */
export function estimateQRVersion(payloadLength: number): number {
  if (payloadLength <= 25) return 1;
  if (payloadLength <= 47) return 2;
  if (payloadLength <= 77) return 3;
  if (payloadLength <= 114) return 4;
  if (payloadLength <= 154) return 5;
  if (payloadLength <= 195) return 6;
  if (payloadLength <= 242) return 7;
  if (payloadLength <= 290) return 8;
  if (payloadLength <= 345) return 9;
  if (payloadLength <= 404) return 10;
  return 11;
}
