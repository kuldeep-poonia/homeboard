# HomeBoard — System Architecture & Design Specification

> Visualized architectural blueprint of **HomeBoard**: an ambient household display engineered for Amazon Fire TV / Vega OS, phone QR pairing, WebSocket live synchronization, and Amazon Bedrock NLP.

---

## 1. High-Level System Architecture

```mermaid
graph TB
    subgraph LivingRoom ["Living Room Environment"]
        TVApp["Fire TV / Vega OS App<br/>(React Native / 10-Foot Web UI)"]
        Remote["Fire TV Remote / D-Pad"]
        Remote -->|"DPAD / Key Events"| TVApp
    end

    subgraph MobileUsers ["Family Members (Mobile Devices)"]
        PhoneA["Phone Browser A<br/>(iOS Safari / Chrome)"]
        PhoneB["Phone Browser B<br/>(Android Chrome)"]
        Camera["Mobile Camera"]
        Camera -->|"Scan QR Code"| PhoneA
    end

    subgraph CloudOrEdgeServer ["HomeBoard Core Platform (Go Backend)"]
        Router["HTTP Router / Middleware<br/>(Security, CORS, Rate Limit)"]
        JoinHandler["Join & Auth Engine<br/>(Token Redemption)"]
        BoardHandler["Board & Items REST API<br/>(Tenant Isolated)"]
        WSHub["Live Sync WebSocket Hub<br/>(Per-Board Client Rooms)"]
        TVHandler["10-Foot Ambient UI Handler<br/>(/tv Dashboard)"]
        
        Router --> TVHandler
        Router --> JoinHandler
        Router --> BoardHandler
        Router --> WSHub
    end

    subgraph DataStorage ["Persistence Layer"]
        DB[("Pure Go SQLite Database<br/>(WAL Mode, SHA-256 Hashed Secrets)")]
    end

    subgraph AIServices ["Natural Language Processing (Dual Engine)"]
        Bedrock["Amazon Bedrock<br/>(Claude 3 Haiku / Nova Micro)"]
        Heuristic["Offline Heuristic Engine<br/>(Multilingual Regex & Rules)"]
    end

    TVApp <-->|"HTTP & Persistent WS<br/>(X-TV-Secret / Query Auth)"| Router
    PhoneA <-->|"HTTP & Ephemeral WS<br/>(Session Cookie)"| Router
    PhoneB <-->|"HTTP & Ephemeral WS<br/>(Session Cookie)"| Router

    JoinHandler <-->|"Verify & Hash Tokens"| DB
    BoardHandler <-->|"Board Scoped Queries"| DB
    BoardHandler -->|"Broadcast Events"| WSHub

    BoardHandler -.->|"Try Cloud LLM"| Bedrock
    BoardHandler -.->|"Fallback on Error / Offline"| Heuristic
```

---

## 2. End-to-End User Flow & Sequence

The diagram below details the complete lifecycle from television initialization and QR pairing to real-time synchronization and D-pad interaction:

```mermaid
sequenceDiagram
    autonumber
    actor User as Family Member
    participant TV as Fire TV (10-Foot UI)
    participant Server as HomeBoard Backend
    participant DB as SQLite DB
    participant WS as WebSocket Hub
    participant AI as Bedrock / Heuristic Engine

    Note over TV,Server: Step 1: Fire TV Board Initialization
    TV->>Server: POST /v1/boards (Bootstrap new board)
    Server->>DB: INSERT INTO boards (id, tv_secret_hash)
    Server-->>TV: 201 Created (board_id, tv_secret)
    TV->>WS: Connect ws://.../v1/boards/{id}/ws?tv_secret={sec}
    WS-->>TV: WebSocket Connected (Live Sync Room)

    Note over TV,Server: Step 2: Dynamic QR Code Generation
    TV->>Server: POST /v1/boards/{id}/join-tokens (Header: X-TV-Secret)
    Server->>DB: Store hashed join token (10-min TTL)
    Server-->>TV: { token: "abc128...", expires_in: 600 }
    TV->>TV: Render QR Code on TV bottom-right corner

    Note over User,TV: Step 3: Phone Camera Scan & Pairing
    User->>TV: Scans QR code with smartphone camera
    User->>Server: GET /j/{token} (Mobile Onboarding Page)
    Server->>DB: Validate token, redeem (Single-Use Mark), Issue Session
    Server-->>User: 200 OK + Set-Cookie (Session Token)

    Note over User,TV: Step 4: Adding Household Note via AI Auto-Categorization
    User->>Server: POST /v1/boards/{id}/items (Text: "Buy fresh milk tomorrow 5pm")
    Server->>AI: ParseText("Buy fresh milk tomorrow 5pm")
    AI-->>Server: { text: "fresh milk", type: "shopping", when: "2026-10-05T17:00:00Z" }
    Server->>DB: INSERT INTO items (board_id, text, type, when_ts)
    Server->>WS: Broadcast { type: "item.created", item: {...} }
    WS-->>TV: Push { type: "item.created" }
    TV->>TV: Instant DOM Update (Card: "Buy", Count: +1)
    Server-->>User: 201 Created

    Note over TV,User: Step 5: TV D-Pad Interaction
    User->>TV: Presses [SELECT / OK] on Fire TV Remote
    TV->>Server: PATCH /v1/boards/{id}/items/{itemId} { done: true }
    Server->>DB: UPDATE items SET done = 1 WHERE id = ? AND board_id = ?
    Server->>WS: Broadcast { type: "item.updated" }
    WS-->>User: Push update to phone
```

---

## 3. Security Architecture & Threat Model

HomeBoard is engineered according to the **OWASP Top 10 (2025)** and **AWS Well-Architected Security Pillar**:

```mermaid
graph LR
    subgraph AttackSurfaces ["Attack Vectors & Mitigations"]
        A["Network Eavesdropping"] -->|"Mitigation"| M1["HTTPS / WSS TLS 1.3 Transport Encryption"]
        B["Physical TV Theft / DB Leak"] -->|"Mitigation"| M2["Zero-Plaintext: SHA-256 Hashes for TV Secrets & Tokens"]
        C["Cross-Tenant Data Leak"] -->|"Mitigation"| M3["Strict Scoping: WHERE id = ? AND board_id = ?"]
        D["Brute Force Token Guessing"] -->|"Mitigation"| M4["Cryptographic 128-bit Entropy + IP Rate Limiting"]
        E["Cross-Site Scripting (XSS)"] -->|"Mitigation"| M5["Safe textContent DOM Injection + Strict CSP Headers"]
        F["Session Hijacking"] -->|"Mitigation"| M6["HttpOnly, SameSite=Strict, Secure Cookie Flags"]
    end
```

### Key Security Safeguards
1. **Zero Plaintext Secrets**: Raw tokens are generated using `crypto/rand`, delivered once to the client, and immediately hashed using `crypto/sha256` before database insertion. Constant-time comparison (`subtle.ConstantTimeCompare`) prevents timing attacks.
2. **Ephemeral Join Tokens**: QR tokens expire within 10 minutes and are single-use. The TV screen automatically rotates the token every 8 minutes.
3. **Tenant Boundary Enforcement**: Boards are logically isolated. Database updates, reads, and deletes require compound primary keys matching both the board identifier and the item identifier.
4. **Device Revocation**: The TV maintains master authorization to revoke any paired smartphone session instantaneously, which immediately severs the active WebSocket connection.

---

## 4. 10-Foot TV UI Component Hierarchy

The TV interface is optimized for television viewing distances (3 meters / 10 feet), featuring a clean, minimalist design with high-contrast elements:

```mermaid
graph TD
    TVRoot["Fire TV 10-Foot Dashboard (/tv)"]
    
    Header["Top Bar (76px)<br/>Brand, Live Sync Dot, Room Name, Clock & Date"]
    MainGrid["4-Column Category Grid<br/>(Minimalist White Theme, 1px Borders)"]
    Footer["Bottom Bar (96px)<br/>D-Pad Remote Hints & Dynamic Pairing QR Card"]
    IdleMode["Ambient Screensaver Overlay<br/>(Triggers after 3m of inactivity)"]

    TVRoot --> Header
    TVRoot --> MainGrid
    TVRoot --> Footer
    TVRoot --> IdleMode

    MainGrid --> Card1["Card 1: Today<br/>Active tasks & reminders"]
    MainGrid --> Card2["Card 2: Upcoming<br/>Scheduled events & calendar"]
    MainGrid --> Card3["Card 3: Buy<br/>Grocery & shopping items"]
    MainGrid --> Card4["Card 4: Family & Movies<br/>Communal watchlist & activities"]

    Card1 --> ItemRow["Interactive TV Item<br/>[Checkbox] [Text] [Type Badge]"]
    ItemRow --> FocusRing["D-Pad Active Focus Ring<br/>(2px solid #111827 outline)"]
```

---

## 5. Entity-Relationship Data Model

The database runs on embedded **pure Go SQLite** (`modernc.org/sqlite`) in Write-Ahead Logging (WAL) mode:

```mermaid
erDiagram
    BOARDS ||--o{ ITEMS : contains
    BOARDS ||--o{ JOIN_TOKENS : generates
    BOARDS ||--o{ DEVICE_SESSIONS : authorizes

    BOARDS {
        TEXT id PK "UUID / Slug"
        TEXT name "Display Room Name"
        TEXT tv_secret_hash "SHA-256 hash of TV secret"
        DATETIME created_at "Timestamp"
        DATETIME updated_at "Timestamp"
    }

    ITEMS {
        TEXT id PK "Item UUID"
        TEXT board_id FK "Board identifier"
        TEXT text "Note text (XSS-safe)"
        TEXT type "reminder | event | shopping | movie | status"
        DATETIME when_ts "Optional parsed deadline / event timestamp"
        INTEGER done "0 = active, 1 = completed"
        INTEGER archived "0 = visible, 1 = archived"
        TEXT created_by_device "Device session identifier"
        DATETIME created_at "Timestamp"
        DATETIME updated_at "Timestamp"
    }

    JOIN_TOKENS {
        TEXT token_hash PK "SHA-256 hash of 128-bit join token"
        TEXT board_id FK "Board identifier"
        DATETIME expires_at "10-minute expiration timestamp"
        INTEGER redeemed "0 = pending, 1 = redeemed"
        DATETIME created_at "Timestamp"
    }

    DEVICE_SESSIONS {
        TEXT session_hash PK "SHA-256 hash of 256-bit session token"
        TEXT board_id FK "Board identifier"
        TEXT device_name "User / Device label"
        INTEGER revoked "0 = active, 1 = revoked"
        DATETIME created_at "Timestamp"
        DATETIME last_seen_at "Heartbeat timestamp"
    }
```

---

## 6. AI Dual-Engine Architecture

HomeBoard implements a fault-tolerant dual natural language engine ensuring 100% operational uptime regardless of cloud availability:

```mermaid
graph TD
    Input["Natural Language Input<br/>'Buy groceries tomorrow at 5pm' or 'doodh lana hai'"]
    CheckCloud{"Bedrock Enabled &amp;<br/>Cloud Accessible?"}
    
    BedrockEngine["Amazon Bedrock Engine<br/>Model: Claude 3 Haiku / Amazon Nova<br/>Region: us-east-1"]
    HeuristicEngine["Built-in Heuristic Engine<br/>Local CPU Regex &amp; Temporal Classifier<br/>Cost: $0.00 / Latency: 0.2ms"]

    StructuredOutput["Structured Result<br/>Type: shopping | Text: groceries | When: 2026-10-05 17:00"]

    Input --> CheckCloud
    CheckCloud -->|Yes| BedrockEngine
    CheckCloud -->|No / Exception / Timeout| HeuristicEngine
    BedrockEngine -->|Success| StructuredOutput
    BedrockEngine -->|API Error / Quota| HeuristicEngine
    HeuristicEngine --> StructuredOutput
```
