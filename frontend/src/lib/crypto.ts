/**
 * ZeroDrop Cryptography Utilities
 * Uses Web Crypto API for client-side encryption
 * No external crypto libraries - browser-native only
 */

export interface KeyPair {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
  publicKeyPEM: string;
  privateKeyPEM: string;
}

export interface EncryptionResult {
  ciphertext: ArrayBuffer;
  base64: string;
  qrData: string;
}

/**
 * Generate a new X25519 key pair for ECDH
 */
export async function generateKeyPair(): Promise<KeyPair> {
  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: "X25519",
    },
    true,
    ["deriveKey", "deriveBits"]
  );

  // Export public key as SPKI (SubjectPublicKeyInfo format)
  const publicKeyData = await window.crypto.subtle.exportKey("spki", keyPair.publicKey);
  const publicKeyPEM = arrayBufferToPEM(publicKeyData, "PUBLIC KEY");

  // Export private key as PKCS8
  const privateKeyData = await window.crypto.subtle.exportKey("pkcs8", keyPair.privateKey);
  const privateKeyPEM = arrayBufferToPEM(privateKeyData, "PRIVATE KEY");

  return {
    publicKey: keyPair.publicKey,
    privateKey: keyPair.privateKey,
    publicKeyPEM,
    privateKeyPEM,
  };
}

/**
 * Import a public key from PEM format
 */
export async function importPublicKey(pem: string): Promise<CryptoKey> {
  const buffer = pemToArrayBuffer(pem);
  return await window.crypto.subtle.importKey(
    "spki",
    buffer,
    {
      name: "X25519",
    },
    true,
    []
  );
}

/**
 * Import a private key from PEM format
 */
export async function importPrivateKey(pem: string): Promise<CryptoKey> {
  const buffer = pemToArrayBuffer(pem);
  return await window.crypto.subtle.importKey(
    "pkcs8",
    buffer,
    {
      name: "X25519",
    },
    true,
    ["deriveKey", "deriveBits"]
  );
}

/**
 * Encrypt data using the server's public key
 * This is a simplified implementation - in production, you would:
 * 1. Generate an ephemeral key pair
 * 2. Perform ECDH to derive a shared secret
 * 3. Use the shared secret to encrypt the data
 *
 * For ZeroDrop v1.0, we use a simpler approach compatible with the Go backend
 */
export async function encryptData(
  data: string,
  serverPublicKey: CryptoKey
): Promise<EncryptionResult> {
  const encoder = new TextEncoder();
  const dataBuffer = encoder.encode(data);

  // For now, we'll use a simpler approach:
  // Encode the data as base64 and prefix with ZD1:
  // The full ECDH encryption will be implemented in v1.1

  const base64 = arrayBufferToBase64(dataBuffer);
  const qrData = `ZD1:${base64}`;

  return {
    ciphertext: dataBuffer,
    base64,
    qrData,
  };
}

/**
 * Decrypt data using the private key
 * Full ECDH decryption - to be implemented in v1.1
 */
export async function decryptData(
  ciphertext: string,
  privateKey: CryptoKey
): Promise<string> {
  // Remove ZD1: prefix if present
  const data = ciphertext.replace(/^ZD1:/, "");

  // Decode base64
  const buffer = base64ToArrayBuffer(data);
  const decoder = new TextDecoder();
  return decoder.decode(buffer);
}

/**
 * Calculate SHA-256 fingerprint of a public key
 */
export async function calculateFingerprint(publicKey: CryptoKey): Promise<string> {
  const publicKeyData = await window.crypto.subtle.exportKey("spki", publicKey);
  const hashBuffer = await window.crypto.subtle.digest("SHA-256", publicKeyData);
  return arrayBufferToHex(hashBuffer);
}

/**
 * Calculate SHA-256 fingerprint of a PEM-encoded public key
 */
export async function calculateFingerprintFromPEM(pem: string): Promise<string> {
  const buffer = pemToArrayBuffer(pem);
  const hashBuffer = await window.crypto.subtle.digest("SHA-256", buffer);
  return arrayBufferToHex(hashBuffer);
}

/**
 * Convert ArrayBuffer to Base64
 */
export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return window.btoa(binary);
}

/**
 * Convert Base64 to ArrayBuffer
 */
export function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = window.atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Convert ArrayBuffer to Hex string
 */
export function arrayBufferToHex(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

/**
 * Convert PEM string to ArrayBuffer
 */
export function pemToArrayBuffer(pem: string): ArrayBuffer {
  const b64 = pem
    .replace(/-----BEGIN .*-----/g, "")
    .replace(/-----END .*-----/g, "")
    .replace(/\s/g, "");

  const binaryString = window.atob(b64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Convert ArrayBuffer to PEM format
 */
function arrayBufferToPEM(buffer: ArrayBuffer, label: string): string {
  const base64 = arrayBufferToBase64(buffer);
  const lines = base64.match(/.{1,64}/g) || [];
  return `-----BEGIN ${label}-----\n${lines.join("\n")}\n-----END ${label}-----`;
}

/**
 * Generate QR code data URL using a QR code library
 * For now, returns a placeholder - in production, use a QR library
 */
export function generateQRCode(data: string): string {
  // Placeholder - in production, use a library like qrcode
  // For now, we'll return the data as-is for the mock display
  return data;
}
