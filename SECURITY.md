# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.1.x   | ✅ Active          |
| 1.0.x   | ❌ End of life     |

## Reporting a Vulnerability

ZeroDrop Terminal takes security seriously. If you discover a security
vulnerability, please report it privately before disclosing it publicly.

**Do NOT report security vulnerabilities via public GitHub issues.**

### How to Report

1. **Email**: Send details to the repository maintainer. If no dedicated
   security email is published, open a GitHub issue with `[SECURITY]` in
   the title and request the maintainer to contact you privately.

2. **What to include**:
   - Type of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

3. **Response timeline**:
   - Acknowledgment within 48 hours
   - Status update within 5 business days
   - Fix timeline communicated based on severity

### What to expect

- You'll receive acknowledgment of your report
- We'll investigate and determine impact
- We'll develop and test a fix
- We'll release a security update and credit you (if desired)

## Scope

The following are IN scope:
- Zero-knowledge guarantee bypass
- Private key extraction or exposure
- Plaintext payload leakage
- Authentication bypass for admin endpoints
- Memory disclosure of sensitive data
- Cryptographic weaknesses

The following are OUT of scope:
- Physical attacks on the thermal printer
- Compromise of the operating system running ZeroDrop
- Social engineering of operators
- Browser-level vulnerabilities (Web Crypto API is trusted)

## Security Features

ZeroDrop Terminal has the following built-in security measures:

- **Zero-knowledge architecture**: Server never possesses plaintext payload
  or private key during operation
- **Burn Protocol**: Private key zeroed from RAM after first-run QR display,
  using `runtime.KeepAlive()` to prevent compiler optimization
- **Persistent key storage**: Private key on disk at 0600 permissions;
  never loaded into server memory on subsequent starts
- **Constant-time comparison**: Admin token comparison uses
  `crypto/subtle.ConstantTimeCompare` to prevent timing attacks
- **Session cookies**: HttpOnly, SameSite=Lax, HMAC-signed, 24-hour expiry
- **Rate limiting**: Per-IP sliding window (5 req/hr default) with separate
  admin login rate limiter (10 attempts/15 min/IP)
- **Input validation**: All external input validated at boundary layer
- **Memory hygiene**: All payload buffers zeroed after print job completion
- **50 automated security checks**: Run `make check-security` to verify
- **No database**: Ephemeral RAM-only processing

## Vulnerability Disclosure Timeline

| Severity | Fix Timeline | Disclosure |
|----------|-------------|------------|
| Critical | 7 days | 30 days after fix |
| High     | 14 days | 30 days after fix |
| Medium   | 30 days | 60 days after fix |
| Low      | 90 days | 90 days after fix |
