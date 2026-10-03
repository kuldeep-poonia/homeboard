# HomeBoard — Security Threat Model & Risk Register

Security is an explicit release gate for HomeBoard. In alignment with OWASP Top 10 (2025), OWASP API Security Top 10, and AWS Well-Architected Security Pillar, every risk is classified with severity and verified mitigations.

## Trust Boundaries
1. **Fire TV Client**: Runs in a semi-public living room environment. Displays board data; must not leak secrets or credentials.
2. **Phone Web Client**: Untrusted device connected via QR scan. Must authenticate using single-use join token redeemed for an opaque session cookie.
3. **Go Backend Service**: Primary policy enforcement and authorization boundary. Never trusts client input.
4. **SQLite Persistence**: Stores only cryptographic hashes of `tv_secret`, `join_token`, and `session_token`. Never raw secrets.
5. **AWS Bedrock**: External inference boundary; all prompt inputs and LLM outputs are treated as untrusted data.

## Threat Register

| ID | Severity | Threat Description | Mitigation Strategy | Verification / Test | Status |
|---|---|---|---|---|---|
| **SEC-01** | **P0** | Plaintext secret exposure in repository or DB | Hash all join tokens (`SHA-256`), device sessions (`SHA-256`), and TV secrets (`Argon2id` / `SHA-256`). Scan via `gitleaks`. | Automated DB inspection test; `gitleaks detect` in CI | Open |
| **SEC-02** | **P0** | Cross-Board Data Exposure (IDOR / BOLA) | Strict board-scoped queries: `WHERE id = ? AND board_id = ?`. Every API request validates caller's board binding. | Automated cross-board authorization test suite | Open |
| **SEC-03** | **P0** | QR Join Token replay or brute force | 128-bit CSPRNG tokens, 10-minute TTL, single-use invalidation immediately upon redemption, rate-limited redeem endpoint. | Token replay & brute-force unit tests | Open |
| **SEC-04** | **P0** | XSS in Phone Web UI | Plain HTML/JS avoids `innerHTML`. All board text rendered strictly via `textContent`. Restrictive Content Security Policy (CSP). | Automated DOM injection test | Open |
| **SEC-05** | **P1** | CSRF on state-changing requests | `SameSite=Lax` / `Strict`, `HttpOnly`, `Secure` cookies, and `Origin`/`Referer` header validation on API endpoints. | CSRF origin mismatch test | Open |
| **SEC-06** | **P1** | Malicious / Oversized payloads (DoS) | Strict request body limits (max 64KB), item text limit (max 200 UTF-8 chars), item type allowlist validation. | Boundary value and oversized payload test | Open |
| **SEC-07** | **P1** | Stale / Revoked phone session abuse | TV-accessible device revocation list. Revocation immediately invalidates session hash and closes open WebSockets. | Revocation flow test | Open |
| **SEC-08** | **P2** | Bedrock Prompt Injection | Input treated strictly as data in prompt template; output validated against strict JSON schema. Fallback to basic text reminder. | Injection sample benchmark test | Open |
