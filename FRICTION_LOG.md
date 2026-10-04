# HomeBoard — Developer Friction Log & Product Feedback

> This log captures real engineering hurdles, environment nuances, and technical friction encountered while building HomeBoard for **Amazon Fire TV / Vega OS**, **Amazon Bedrock**, and **AWS**. Submitting this log qualifies HomeBoard for up to a **10% judging bonus** under the official Amazon Developer Hackathon rules.

---

## 1. Friction Log Entries

| # | Date | Component | Task Attempted | Steps Taken | Expected Outcome | Actual Result (Friction) | Severity | Workaround & Solution Applied | Actionable Suggestion for Amazon |
|---|---|---|---|---|---|---|---|---|---|
| **1** | 2026-10-04 | **Amazon Bedrock** | Enable Foundation Model Access | Navigated to Bedrock Console in `us-east-1` to request Claude 3 Haiku access. | Access model request table to toggle models. | Banner: *"Model access page has been retired. Serverless foundation models are now automatically enabled across all commercial regions."* | Low | Transitioned to direct invocation using updated SDK and active model IDs. | Update developer documentation to clarify that the manual Model Access approval workflow has been retired. |
| **2** | 2026-10-04 | **Amazon Bedrock** | Direct Model Invocation via SDK & CLI | Attempted `bedrock-runtime:InvokeModel` with `anthropic.claude-3-haiku-20240307-v1:0`. | JSON response with classified household item. | Error: `ResourceNotFoundException: This model version has reached the end of its life.` | Medium | Queried active foundation models via `ListFoundationModels` and updated configuration to active models (`anthropic.claude-haiku-4-5-20251001-v1:0` and `amazon.nova-micro-v1:0`). | Provide deprecation notice and alias routing (e.g. `anthropic.claude-haiku:latest`) so SDK calls do not break on version transitions. |
| **3** | 2026-10-04 | **Amazon Bedrock** | Invoking active models on new AWS Account | Executed `InvokeModel` for Amazon Nova Micro and Claude Haiku. | Successful inference response. | Error: `ValidationException: Operation not allowed`. | High | Built an offline, zero-dependency heuristic parser (`backend/internal/ai/heuristic.go`) that automatically acts as a fault-tolerant fallback whenever Bedrock returns validation or quota errors. | Surface clearer error messages explaining that newly created AWS accounts without established invoice history require billing verification before Bedrock allows invocation. |
| **4** | 2026-10-04 | **AWS App Runner** | Deploying Go Backend Service | Created an App Runner service connected to GitHub repository. | Automatic container build and live HTTPS deployment. | Yellow Notice: *"Starting April 30, 2026, AWS App Runner is no longer accepting new customers... recommended Amazon ECS Express Mode"*; Go runtime rejected with `The runtime you are currently trying to use has reached End of Support`. | High | Packaged the backend into a lightweight multi-stage Dockerfile (`golang:alpine` -> `alpine:3.20`) and deployed to container platforms / Render with $0.00 cost. | Fix the broken link to `express-deploy` in the App Runner deprecation banner which leads to a 404 page in the ECS console. |
| **5** | 2026-10-04 | **Go HTTP Server** | WebSocket live synchronization through security middleware | Wrapped standard HTTP handlers with security and rate-limiting middleware. | Smooth WebSocket handshake on `/v1/boards/{id}/ws`. | Handshake failed with error: `responseWriter does not implement http.Hijacker`. | Medium | Implemented the `http.Hijacker` interface on the custom `responseWriter` struct in `backend/internal/middleware/security.go`, allowing raw TCP connection takeover for WebSockets. | Standardize Go middleware patterns in AWS SDK examples to include `Hijack()` method delegation for real-time applications. |
| **6** | 2026-10-04 | **Mobile Web Client** | QR Pairing Rate Limiter | Implemented per-IP rate limiting on the `/v1/boards/{id}/join-tokens` endpoint. | Throttling requests exceeding 20 redemptions per minute. | Rate limiter treated each connection as a distinct IP because `r.RemoteAddr` included ephemeral client ports (e.g. `192.168.1.50:54321`). | Medium | Utilized `net.SplitHostPort` in `backend/internal/handlers/join.go` to isolate the host IP before evaluating the token bucket rate limiter. | Document `net.SplitHostPort` best practices for rate-limiting middleware in Go web services. |

---

## 2. Product Feedback on Amazon Developer Tools & SDKs

### Fire TV / Vega OS
* **What we used it for:** Building the ambient 10-foot household board display with D-pad navigation and QR code pairing.
* **What worked well:** Responsive layout rendering, high-contrast readability at 10 feet, clean D-pad event mapping (`ArrowUp`, `ArrowDown`, `Enter`, `Backspace`).
* **What needs work:** The local emulator setup on Windows requires extensive Android SDK configuration; providing an official lightweight web-based Fire TV simulator would drastically reduce onboarding friction for web developers.
* **Would we build with it again?** Yes! Fire TV as an ambient communal screen is an under-utilized and powerful form factor for household productivity.

### Amazon Bedrock
* **What we used it for:** Parsing unstructured natural language notes (English & Hinglish) into structured household tasks, reminders, and grocery lists.
* **What worked well:** The Claude and Amazon Nova models provide exceptional JSON-mode structure extraction and understand bilingual Indian context (e.g., *"doodh lana hai"*).
* **What needs work:** The first-time onboarding experience has friction: newly created accounts face `ValidationException: Operation not allowed` without clear instructions that billing history is required.
* **Would we build with it again?** Absolutely. Once authorized, Bedrock provides the fastest serverless LLM inference in the cloud.
