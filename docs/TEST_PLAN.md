# HomeBoard — Test Plan & Execution Matrix

This document tracks test scenarios and exit gates across all implementation phases for HomeBoard.

## Primary Test Environments
- **Fire TV Client**: Vega TV Simulator / Fire TV React Native environment.
- **Phone Client**: Mobile browser (Chrome Android / Safari iOS) via responsive web interface.
- **Backend**: Go `net/http` service on `http://localhost:8080` (Development) / HTTPS (Production).

## Test Matrix by Phase

| Phase | Description | Key Gate / Criteria | Status |
|---|---|---|---|
| **Phase 0** | Setup & Skeleton | Simulator launch, `/healthz` 200 OK, clean Go/TS build, clean `.gitignore` | Verified Pass |
| **Phase 1** | Backend Core | SQLite migrations, Board creation, Items CRUD, 200 char limit, tenant isolation | Verified Pass |
| **Phase 2** | Pairing & Sessions | 128-bit join token, SHA-256 hashing, 10m TTL, one-time use, device cookie session | Verified Pass |
| **Phase 3** | Phone Web UI | Clean mobile web `/j/{token}`, add/edit/done items, XSS prevention, responsive UI | Verified Pass |
| **Phase 4** | Fire TV App Board UI | 10-foot UI, D-pad navigation, persistent corner QR, automatic token refresh | Verified Pass |
| **Phase 5** | WebSocket Live Sync | Live push notifications (<1s), board-scoped rooms, auto-reconnect with jitter | Verified Pass |
| **Phase 6** | Remote Control & Idle | D-pad shortcuts (OK=done, Long-press=archive), foreground ambient idle mode | Verified Pass |
| **Phase 7** | Amazon Bedrock AI | Natural language parse to structured board items, 100% benchmark, strict fallback | Verified Pass |
| **Phase 8** | Security Hardening | OWASP Top 10 verification, constant-time comparisons, rate limiting, zero plaintext tokens | Verified Pass |
| **Phase 9** | Deployment | Multi-stage Docker, AWS App Runner & Compose configs, HTTPS, zero repo secrets | Verified Pass |
| **Phase 10** | Fire TV Android Build | Android TV Leanback launcher, D-pad manifests, Gradle configs | Ready |

## Automated Verification Commands
```bash
# Backend checks
cd backend
go vet ./...
go test -v -race ./...

# Frontend type checking
cd tv-app
npm run typecheck
```
