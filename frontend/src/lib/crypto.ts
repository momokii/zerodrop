/**
 * ZeroDrop Cryptography Utilities
 * Uses Web Crypto API for client-side ECIES encryption (X25519 ECDH + AES-256-GCM)
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
    { name: "X25519" },
    true,
    ["deriveKey", "deriveBits"]
  );

  const publicKeyData = await window.crypto.subtle.exportKey("spki", keyPair.publicKey);
  const publicKeyPEM = arrayBufferToPEM(publicKeyData, "PUBLIC KEY");

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
  const buffer = parsePEM(pem);
  return await window.crypto.subtle.importKey(
    "spki",
    buffer,
    { name: "X25519" },
    true,
    []
  );
}

/**
 * Import a private key from PEM format
 */
export async function importPrivateKey(pem: string): Promise<CryptoKey> {
  const buffer = parsePEM(pem);
  return await window.crypto.subtle.importKey(
    "pkcs8",
    buffer,
    { name: "X25519" },
    true,
    ["deriveKey", "deriveBits"]
  );
}

/**
 * Encrypt data using the server's public key with ECIES (X25519 ECDH + AES-256-GCM)
 *
 * Protocol:
 * 1. Generate ephemeral X25519 key pair
 * 2. ECDH derive shared secret with server's public key
 * 3. Use shared secret as AES-256-GCM key
 * 4. Encrypt plaintext with random 12-byte IV
 * 5. Payload: ZD1:base64(rawEphPubKey(32) + iv(12) + aesCiphertextWithTag)
 */
export async function encryptData(
  data: string,
  serverPublicKey: CryptoKey
): Promise<EncryptionResult> {
  const encoder = new TextEncoder();
  const dataBuffer = encoder.encode(data);

  // Generate ephemeral X25519 key pair
  const ephKeyPair = await window.crypto.subtle.generateKey(
    { name: "X25519" },
    true,
    ["deriveBits"]
  );

  // ECDH: derive shared secret (256 bits / 32 bytes)
  const sharedSecret = await window.crypto.subtle.deriveBits(
    { name: "X25519", public: serverPublicKey },
    ephKeyPair.privateKey,
    256
  );

  // Import shared secret as AES-256-GCM key
  const aesKey = await window.crypto.subtle.importKey(
    "raw",
    sharedSecret,
    { name: "AES-GCM" },
    false,
    ["encrypt"]
  );

  // Generate random 12-byte IV
  const iv = window.crypto.getRandomValues(new Uint8Array(12));

  // Encrypt with AES-256-GCM (returns ciphertext + 16-byte auth tag)
  const encrypted = await window.crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    aesKey,
    dataBuffer
  );

  // Export ephemeral public key as raw 32 bytes
  const ephPubRaw = await window.crypto.subtle.exportKey("raw", ephKeyPair.publicKey);

  // Combine: ephPubKey(32) + iv(12) + ciphertextWithTag(variable)
  const combined = new Uint8Array(32 + 12 + encrypted.byteLength);
  combined.set(new Uint8Array(ephPubRaw), 0);
  combined.set(iv, 32);
  combined.set(new Uint8Array(encrypted), 44);

  const base64 = arrayBufferToBase64(combined.buffer);
  const qrData = `ZD1:${base64}`;

  return {
    ciphertext: combined.buffer,
    base64,
    qrData,
  };
}

/**
 * Decrypt data using the private key (ECIES: X25519 ECDH + AES-256-GCM)
 *
 * Reverse of encryptData:
 * 1. Strip ZD1: prefix, base64 decode
 * 2. Extract: ephPubKey(32) + iv(12) + ciphertextWithTag
 * 3. ECDH derive shared secret
 * 4. Decrypt with AES-256-GCM
 */
export async function decryptData(
  ciphertext: string,
  privateKey: CryptoKey
): Promise<string> {
  // Remove ZD1: prefix
  const data = ciphertext.replace(/^ZD1:/, "");
  const raw = base64ToArrayBuffer(data);
  const rawBytes = new Uint8Array(raw);

  if (rawBytes.length < 44) {
    throw new Error("Invalid payload: too short");
  }

  // Extract components
  const ephPubKeyRaw = rawBytes.slice(0, 32);
  const iv = rawBytes.slice(32, 44);
  const ciphertextWithTag = rawBytes.slice(44);

  // Import ephemeral public key from raw bytes
  const ephPubKey = await window.crypto.subtle.importKey(
    "raw",
    ephPubKeyRaw,
    { name: "X25519" },
    false,
    []
  );

  // ECDH derive shared secret
  const sharedSecret = await window.crypto.subtle.deriveBits(
    { name: "X25519", public: ephPubKey },
    privateKey,
    256
  );

  // Import as AES-GCM key
  const aesKey = await window.crypto.subtle.importKey(
    "raw",
    sharedSecret,
    { name: "AES-GCM" },
    false,
    ["decrypt"]
  );

  // AES-GCM decrypt (validates auth tag automatically)
  const plaintextBytes = await window.crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    aesKey,
    ciphertextWithTag
  );

  const decoder = new TextDecoder();
  return decoder.decode(plaintextBytes);
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
  const buffer = parsePEM(pem);
  const hashBuffer = await window.crypto.subtle.digest("SHA-256", buffer);
  return arrayBufferToHex(hashBuffer);
}

/**
 * Parse a PEM string to ArrayBuffer (strips headers and decodes base64)
 */
export function parsePEM(pem: string): ArrayBuffer {
  const b64 = pem
    .replace(/-----BEGIN [\w\s]+-----/g, "")
    .replace(/-----END [\w\s]+-----/g, "")
    .replace(/\s/g, "");
  const binary = window.atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
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
 * Convert ArrayBuffer to PEM format
 */
function arrayBufferToPEM(buffer: ArrayBuffer, label: string): string {
  const base64 = arrayBufferToBase64(buffer);
  const lines = base64.match(/.{1,64}/g) || [];
  return `-----BEGIN ${label}-----\n${lines.join("\n")}\n-----END ${label}-----`;
}
