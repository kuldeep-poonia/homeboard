# HomeBoard — Test Plan & Execution Matrix

This document tracks test scenarios and exit gates across all implementation phases for HomeBoard.

## Primary Test Environments
- **Fire TV Client**: Vega TV Simulator / Fire TV React Native environment.
- **Phone Client**: Mobile browser (Chrome Android / Safari iOS) via responsive web interface.
- **Backend**: Go `net/http` service on `http://localhost:8080` (Development) / HTTPS (Production).

## Test Matrix by Phase

| Phase | Description | Key Gate / Criteria | Status |
|---|---|---|---|
| **Phase 0** | Setup & Skeleton | Simulator launch, `/healthz` 200 OK, clean Go/TS build, clean `.gitignore` | In Progress |
| **Phase 1** | Backend Core | SQLite migrations, Board creation, Items CRUD, 200 char limit, tenant isolation | Pending |
| **Phase 2** | Pairing & Sessions | 128-bit join token, SHA-256 hashing, 10m TTL, one-time use, device cookie session | Pending |
| **Phase 3** | Phone Web UI | Clean mobile web `/j/{token}`, add/edit/done items, XSS prevention, responsive UI | Pending |
| **Phase 4** | Fire TV App Board UI | 10-foot UI, D-pad navigation, persistent corner QR, automatic token refresh | Pending |
| **Phase 5** | WebSocket Live Sync | Live push notifications (<1s), board-scoped rooms, auto-reconnect with jitter | Pending |
| **Phase 6** | Remote Control & Idle | D-pad shortcuts (OK=done, Long-press=archive), foreground ambient idle mode | Pending |
| **Phase 7** | Amazon Bedrock AI | Optional natural language parse to structured board items, strict fallback | Pending |
| **Phase 8** | Security Hardening | OWASP Top 10 verification, constant-time comparisons, rate limiting, zero plaintext tokens | Pending |
| **Phase 9** | Deployment | AWS single-instance deployment with HTTPS, zero repo secrets | Pending |

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
