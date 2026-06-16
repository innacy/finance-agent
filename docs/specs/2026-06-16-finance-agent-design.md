# Finance Agent — Design Specification

**Date**: 2026-06-16
**Status**: Approved
**Stack**: Go 1.22+, MongoDB, Gmail API, NVIDIA AI

## Overview

A CLI-based personal finance agent styled after the gitlab-cli REPL. It ingests financial data from Gmail bank alerts and PDF statements (HDFC Bank initially), consolidates transactions, auto-categorizes spending, and presents a CRED "My Money"-style view of accounts, cards, and spending patterns.

### Goals

- Consolidated view of all bank accounts and credit cards
- Automatic transaction ingestion from Gmail alerts
- Smart categorization (rules-first, AI fallback via NVIDIA)
- Monthly spend analysis by category
- Background daemon for continuous email polling
- Data stored in MongoDB for future iData-UI consumption

### Non-Goals (v0.1)

- Investments (MF, stocks, FD/RD) — deferred to v0.2
- Broker API integrations
- Multi-user support
- Web/mobile UI (consumed via iData-UI later)

---

## Architecture

### Layered Design (mirroring gitlab-cli)

```
REPL (cmd/) — UX, prompts, display, dispatch
    ↓
Pkg Layer — reusable business logic
    ├── pkg/sources/     — data source abstraction (Gmail, Statement)
    ├── pkg/parsers/     — bank-specific email/PDF parsing
    ├── pkg/categorization/ — rule engine + AI fallback
    ├── pkg/db/          — MongoDB client and collection ops
    ├── pkg/ai/          — NVIDIA AI client
    ├── pkg/config/      — Viper config loading
    └── pkg/output/      — terminal theming, tables, boxes
    ↓
internal/models/ — shared domain types (not exported)
    ↓
MongoDB — persistent storage
```

### Project Structure

```
finance-agent/
├── main.go
├── go.mod / go.sum
├── Makefile
├── README.md
├── config.yaml                     # Local secrets (gitignored)
├── configs/default.yaml            # Config template
│
├── cmd/
│   ├── repl.go                     # Core loop, dispatch, session, completer
│   ├── repl_account.go             # Account commands
│   ├── repl_txn.go                 # Transaction commands
│   ├── repl_card.go                # Credit card commands
│   ├── repl_sync.go                # Sync commands
│   ├── repl_overview.go            # Overview & analytics
│   ├── repl_category.go            # Category management
│   ├── repl_config.go              # Config display
│   ├── repl_helpers.go             # Shared utilities
│   ├── repl_prompts.go             # Interactive prompt helpers
│   └── daemon.go                   # Background polling mode
│
├── pkg/
│   ├── config/
│   ├── db/
│   │   ├── mongo.go
│   │   ├── accounts.go
│   │   ├── transactions.go
│   │   ├── cards.go
│   │   └── categories.go
│   ├── sources/
│   │   ├── source.go               # Source interface
│   │   ├── gmail/
│   │   └── statement/
│   ├── parsers/
│   │   ├── parser.go               # Parser interface
│   │   └── hdfc/
│   │       ├── email.go
│   │       ├── statement.go
│   │       └── templates.go
│   ├── categorization/
│   │   ├── engine.go               # Multi-layer pipeline orchestrator
│   │   ├── memory.go               # Layer 1: Merchant memory (exact match)
│   │   ├── fuzzy.go                # Layer 2: Fuzzy string matching
│   │   ├── classifier.go           # Layer 3: Naive Bayes ML classifier
│   │   ├── patterns.go             # Layer 4: Recurring/time/amount heuristics
│   │   ├── ai.go                   # Layer 5: NVIDIA AI fallback
│   │   ├── training.go             # Training data management & retraining
│   │   └── seeds.go                # Built-in merchant seed data
│   ├── ai/
│   ├── output/
│   └── utils/
│
├── internal/models/
│   ├── account.go
│   ├── transaction.go
│   ├── card.go
│   ├── category.go
│   └── overview.go
│
└── docs/
    └── specs/
```

---

## Data Model

### Collection: `accounts`

| Field | Type | Description |
|-------|------|-------------|
| `_id` | ObjectID | Auto-generated |
| `user_id` | string | Owner identifier |
| `bank_name` | string | "HDFC" |
| `account_number` | string | Last 4 digits |
| `account_type` | string | savings, current, salary |
| `balance` | float64 | Current balance |
| `currency` | string | "INR" |
| `last_updated` | time | Last balance update |
| `is_active` | bool | Active flag |
| `is_confirmed` | bool | User confirmed tracking (auto-detect flow) |
| `detected_from` | string | Source that first detected this account |
| `metadata` | map | IFSC, branch, etc. |

**Unique index**: `(user_id, bank_name, account_number)`

### Collection: `transactions`

| Field | Type | Description |
|-------|------|-------------|
| `_id` | ObjectID | Auto-generated |
| `user_id` | string | Owner |
| `account_id` | ObjectID | Linked account |
| `type` | string | debit, credit |
| `amount` | float64 | Transaction amount |
| `balance_after` | float64 | Balance post-transaction |
| `description` | string | Raw bank description |
| `merchant` | string | Extracted merchant name |
| `category` | string | food, travel, bills, etc. |
| `sub_category` | string | restaurant, fuel, etc. |
| `tags` | []string | User-defined tags |
| `transaction_date` | time | When it occurred |
| `value_date` | time | Value date |
| `reference` | string | UPI ref / cheque number |
| `channel` | string | UPI, NEFT, IMPS, ATM, POS |
| `counterparty_upi` | string | UPI ID if available |
| `source` | string | "gmail_alert", "statement_pdf" |
| `source_ref` | string | Email message ID / file hash |
| `categorized_by` | string | "rule", "ai", "manual" |
| `is_recurring` | bool | Recurring transaction flag |
| `created_at` | time | Record creation time |

**Indexes**: `(user_id, transaction_date)`, `(user_id, category)`, `(user_id, account_id, transaction_date)`

### Collection: `credit_cards`

| Field | Type | Description |
|-------|------|-------------|
| `_id` | ObjectID | Auto-generated |
| `user_id` | string | Owner |
| `bank_name` | string | Issuing bank |
| `card_name` | string | "HDFC Regalia" |
| `card_number` | string | Last 4 digits |
| `credit_limit` | float64 | Total limit |
| `available_limit` | float64 | Available limit |
| `outstanding` | float64 | Current outstanding |
| `minimum_due` | float64 | Minimum payment |
| `due_date` | time | Payment due date |
| `statement_date` | time | Statement generation |
| `billing_cycle` | int | Day of month |
| `utilization` | float64 | Usage percentage |
| `last_updated` | time | Last update |
| `is_active` | bool | Active flag |

**Unique index**: `(user_id, card_number)`

### Collection: `categories`

| Field | Type | Description |
|-------|------|-------------|
| `_id` | ObjectID | Auto-generated |
| `user_id` | string | Owner |
| `category` | string | Category name |
| `sub_category` | string | Sub-category |
| `match_type` | string | merchant, upi_id, keyword, regex |
| `pattern` | string | Match pattern |
| `priority` | int | Higher = checked first |
| `is_system` | bool | Built-in vs user-created |
| `created_at` | time | Creation time |

**Index**: `(user_id, priority)` descending

### Collection: `sync_state`

| Field | Type | Description |
|-------|------|-------------|
| `_id` | ObjectID | Auto-generated |
| `user_id` | string | Owner |
| `source` | string | "gmail", "statement" |
| `last_sync_time` | time | Last successful sync |
| `last_message_id` | string | Gmail message ID bookmark |
| `total_processed` | int64 | Lifetime count |
| `last_error` | string | Last error message |
| `status` | string | idle, syncing, error |

**Unique index**: `(user_id, source)`

### Default Categories (seeded)

| Category | Sub-categories |
|----------|---------------|
| Food & Dining | Restaurants, Groceries, Food delivery |
| Transport | Fuel, Cab/Auto, Public transport |
| Shopping | Online, Clothing, Electronics |
| Bills & Utilities | Electricity, Water, Gas, Internet, Mobile |
| Entertainment | Movies, Streaming, Gaming |
| Health | Medical, Pharmacy, Gym |
| Transfers | Self-transfer, Sent to contacts |
| Income | Salary, Freelance, Refunds, Interest |
| EMI & Loans | Home loan, Personal loan, Car loan |
| Subscriptions | Monthly recurring services |
| Others | Uncategorized |

---

## CLI Commands

### Session

| Command | Description |
|---------|-------------|
| `start` | Initialize session, connect DB, verify Gmail auth |
| `config` | Show current configuration |
| `help` | Print command reference |
| `exit` | End session with summary |

### Accounts (`repl_account.go`)

| Command | Description |
|---------|-------------|
| `accounts` | List all bank accounts with balances |
| `account-add` | Add a new bank account interactively |
| `account-update` | Update account details |
| `account-remove` | Deactivate an account |
| `balance` | Quick total balance across all accounts |

### Transactions (`repl_txn.go`)

| Command | Description |
|---------|-------------|
| `txns` | List recent transactions (last 30 days) |
| `txn-search` | Search by keyword, amount, date range |
| `txn-categorize` | Manually categorize a transaction |
| `txn-tag` | Add/remove tags |
| `txn-recurring` | Show detected recurring transactions |

### Credit Cards (`repl_card.go`)

| Command | Description |
|---------|-------------|
| `cards` | List credit cards with outstanding/limits |
| `card-add` | Add a new credit card |
| `card-bill` | Current billing details |
| `card-spend` | Spend breakdown for billing cycle |
| `card-due` | Upcoming due dates |

### Sync (`repl_sync.go`)

| Command | Description |
|---------|-------------|
| `sync` | Pull new Gmail emails, parse & store |
| `sync-status` | Last sync time, counts, errors |
| `sync-history` | Sync run history |
| `import` | Import PDF/CSV statement |

### Overview (`repl_overview.go`)

| Command | Description |
|---------|-------------|
| `overview` | All accounts summary, net position, dues |
| `monthly` | Income vs spend for current month |
| `spend` | Category-wise spend breakdown |
| `spend-trend` | Month-over-month comparison |

### Categories (`repl_category.go`)

| Command | Description |
|---------|-------------|
| `categories` | List categories with counts |
| `category-add` | Add categorization rule |
| `category-edit` | Modify rule |
| `category-remove` | Remove rule |
| `recategorize` | Re-run engine on uncategorized txns |

### Daemon

| Command | Description |
|---------|-------------|
| `daemon-start` | Start background polling from REPL |
| `daemon-stop` | Stop background polling |
| `daemon-status` | Show daemon state |

### Natural Language Fallback

Unrecognized input → NVIDIA AI intent classification → route to command or free-form answer.

---

## Data Ingestion Pipeline

### Gmail Source

**Auth**: OAuth2 with offline refresh token.

**Authentication lifecycle**:

1. **First run**: `gmail-auth` command opens browser for Google consent. User grants Gmail read access. Agent receives access token + refresh token, stores both in `token.json`.
2. **Subsequent runs**: Agent loads `token.json`, uses refresh token to silently obtain a new access token. No browser interaction needed. Token refresh happens automatically before expiry.
3. **Token revoked/expired**: If refresh token is invalid (user revoked in Google settings, or token expired after 6 months of inactivity), agent catches the error and prompts: "Gmail auth expired. Re-authenticate? [y/n]" → re-runs browser OAuth flow.
4. **Auth failure (network, denied, etc.)**: Agent starts in offline mode. REPL commands for viewing stored data work normally. Only sync commands (`sync`, `daemon-start`) fail with: "Gmail unavailable — working in offline mode. Run `gmail-auth` to reconnect."
5. **Daemon mode auth failure**: Logs error, retries next poll cycle. After 3 consecutive failures, pauses polling and logs warning.

**Initial Sync (first run)**:
- During `start` / `gmail-auth`, prompt user: "Sync emails from when? [1 month / 3 months / 6 months / 1 year / all]"
- Fetch all matching bank emails from chosen date onwards
- Process in batches (50 at a time) to avoid memory pressure
- Store the oldest processed timestamp as baseline in `sync_state`

**Subsequent Syncs (incremental)**:
- Query Gmail API with: `from:(alerts@hdfcbank.net) after:<last_sync_timestamp>`
- Only fetches emails newer than `sync_state.last_sync_time`
- Daemon: polls every 5 minutes (configurable)
- REPL `sync`: immediate incremental pull
- If sync gap > 7 days (e.g. agent was offline), warn user and offer backfill

**Processing flow**:

```
Gmail API → Fetch bank emails
    → Identify bank (from address)
    → Route to bank parser
    → Extract transaction data
    → Auto-detect account/card (see below)
    → Deduplicate (reference + amount + date)
    → Run categorization
    → Store in MongoDB
    → Update account balance
```

### Account & Card Auto-Detection

Bank accounts and credit cards are **not** pre-configured — they are discovered from email data:

**Auto-detection flow**:
```
Email parsed → contains account number (last 4 digits) + bank name
    → Check if account exists in DB
    → If new account found:
        → Prompt user: "Detected HDFC Savings ****4521. Track this account? [y/n]"
        → If yes: create account record, set initial balance from email
        → If no: add to ignore list (won't prompt again)
    → If known account: update balance from email
```

**Same for credit cards**:
```
Card transaction email → contains card last 4 digits
    → "Detected HDFC Credit Card ****7788. Track this card? [y/n]"
    → If yes: create card record
```

**Manual override**: Users can also `account-add` / `card-add` manually (for accounts that don't have email alerts enabled yet, or to pre-configure before first sync).

**Daemon mode auto-detect**: In headless daemon mode, new accounts are auto-created with `is_confirmed: false` flag. On next REPL session, user is prompted to confirm/reject pending accounts.

### HDFC Email Templates

| Email Type | Subject Pattern | Extracted Fields |
|-----------|----------------|------------------|
| UPI Debit | "Alert : Update for your HDFC Bank A/c" | amount, UPI ref, merchant/VPA, balance |
| UPI Credit | Same, "credited" in body | amount, sender VPA, balance |
| NEFT/IMPS Credit | "credited to a/c" | amount, sender, ref, balance |
| ATM Withdrawal | "withdrawn from ATM" | amount, location, balance |
| POS/Card Txn | "used at" | amount, merchant, card last 4 |
| Credit Card Alert | "HDFC Bank Credit Card" | amount, merchant, card |
| Bill Generated | "Credit Card Statement" | total due, min due, due date |

### PDF Statement Parser

- Extract text via `pdfcpu` or `unidoc`
- Parse tabular rows (date, narration, debit/credit, balance)
- Match against existing transactions to fill gaps
- Flag unmatched as new entries

### Deduplication

1. Primary key: `(account_id, transaction_date, amount, reference)`
2. Fallback: `(account_id, transaction_date, amount, description_hash)`
3. Statement imports mark matches as `source: "statement_verified"`

---

## Categorization Engine (The Learning Brain)

The categorization engine is a multi-layer pipeline. Each layer is faster and cheaper
than the next. As the system learns from user behavior, more transactions are caught
by the upper (cheaper) layers, and the expensive AI fallback diminishes over time.

### Multi-Layer Pipeline

```
Layer 1: Merchant Memory     [instant, O(1) lookup]
    ↓ no match
Layer 2: Fuzzy Match         [~1ms, string similarity]
    ↓ no match
Layer 3: Local ML Classifier [~5ms, no API, trained on YOUR data]
    ↓ low confidence
Layer 4: Pattern Detection   [heuristics: recurring, time-based, amount-based]
    ↓ no match
Layer 5: NVIDIA AI Fallback  [expensive, last resort — decreases over time]
```

### Layer 1: Merchant Memory (exact match)

Stored in MongoDB, cached in memory at startup. Every confirmed categorization adds an entry.

```go
type MerchantMemory struct {
    entries map[string]string  // normalized description → category
    upiMap  map[string]string  // UPI ID → category
}
```

After 3 months of use, catches 70-80% of transactions alone.

### Layer 2: Fuzzy Match (string similarity)

Handles merchant name variations:
- "SWIGGY BANGALORE", "SWIGGY MUMBAI", "SWIGGY 12345" all match known "SWIGGY"
- Levenshtein distance / token overlap
- Threshold: 85% similarity → same category as matched merchant

### Layer 3: Local Naive Bayes Classifier (the core brain)

TF-IDF Naive Bayes classifier trained on user's own transaction history.

**Library**: [`jbrukh/bayesian`](https://github.com/jbrukh/bayesian) (812 stars, Go native, TF-IDF, persistence)

```go
type LocalClassifier struct {
    model      *bayesian.Classifier
    categories []bayesian.Class
    dataFile   string  // persisted model file
}

func (c *LocalClassifier) Predict(description string) (category string, confidence float64)
func (c *LocalClassifier) Train(description string, category string, weight float64)
func (c *LocalClassifier) Retrain(allData []TrainingEntry)
```

- Retrained incrementally on every confirmation/correction
- After ~200 transactions, reaches 85-90% accuracy for unknowns
- No external API needed — runs in-process in Go

### Layer 4: Pattern Detection (heuristics)

- Same amount + same merchant + regular interval → recurring (subscription/EMI)
- Salary-day credits (same day each month, large amount) → Income
- ATM round amounts → Cash Withdrawal
- Channel-based: NEFT on salary day → Income

### Layer 5: NVIDIA AI Fallback (diminishing use)

- Only triggered when local ML confidence < `categories.min_confidence` (default 0.7)
- Batch up to 10 txns per API call
- After AI categorizes → feeds back into Layer 1 + Layer 3 training data
- Prompt: "Should I remember: DECATHLON → Shopping? [y/n]" → creates persistent entry

### Confidence Scoring

| Source | Confidence | User Action |
|--------|-----------|-------------|
| Merchant Memory (exact) | 0.99 | Auto-accept, no prompt |
| Fuzzy Match | 0.85-0.95 | Auto-accept, no prompt |
| Local ML (high) | 0.80-0.95 | Auto-accept, shown in sync output |
| Local ML (medium) | 0.60-0.80 | Accept but flag for review |
| AI fallback | 0.70-0.90 | Accept, prompt to create rule |
| No match | 0.0 | Mark "Uncategorized", ask user |

### Training Data

```go
type TrainingEntry struct {
    Description string
    Category    string
    Weight      float64   // corrections=10x, confirms=1x, AI=0.5x, seed=0.3x
    Source      string    // "user_correct", "user_confirm", "ai_accepted", "seed"
    CreatedAt   time.Time
}
```

### Built-in Seed Data (~300 Indian merchants)

Shipped with binary for day-0 functionality:

| Pattern | Category |
|---------|----------|
| swiggy, zomato, dominos | Food & Dining / Delivery |
| uber, ola, rapido | Transport / Cab |
| amazon, flipkart, myntra | Shopping / Online |
| netflix, hotstar, spotify | Entertainment / Streaming |
| NEFT-SALARY, SAL- | Income / Salary |
| LIC, insurance | Bills / Insurance |
| electricity, BESCOM, MSEDCL | Bills / Electricity |
| phonepe, gpay, paytm (transfer) | Transfers |

### Retraining Strategy

| Trigger | Action |
|---------|--------|
| Every 50 new confirmed transactions | Incremental retrain |
| User runs `recategorize` | Full retrain + re-predict uncategorized |
| Monthly (daemon mode) | Full retrain on all accumulated data |
| User corrects a prediction | Immediate incremental update |

### Expected Improvement Timeline

```
Week 1:   ~40% auto-categorized (seeds + first confirmations)
Week 2:   ~60% (merchant memory growing, ML has ~100 examples)
Month 1:  ~80% (ML confident, most regular merchants known)
Month 3:  ~92% (fuzzy catches variations, patterns detected)
Month 6:  ~97% (AI barely needed, only truly new merchants)
```

### User Overrides

- `txn-categorize` — manually set category → feeds merchant memory + training data (10x weight)
- Manual categorization always wins over all layers
- `recategorize` — re-runs full pipeline on uncategorized transactions

### Open Source References

| Project | Architecture | URL |
|---------|-------------|-----|
| **Spectra** | Memory → Fuzzy → TF-IDF+LogReg → API. Corrections 10x weight. | [github](https://github.com/francescogabrieli/Spectra) |
| **FafyCat** | Local ML, active learning, >90% accuracy | [github](https://github.com/davidchris/fafycat) |
| **NumbyAI** | Rules → Local LLM (Ollama), batch processing | [github](https://github.com/RoXsaita/NumbyAI-Public) |
| **MoneyPulse** | 60+ seed rules, learning engine, local Ollama | [github](https://github.com/ManikantaR/MoneyPulse) |
| **jbrukh/bayesian** | Go Naive Bayes + TF-IDF (core ML library) | [github](https://github.com/jbrukh/bayesian) |

---

## Daemon Mode

### Run Modes

```
finance-agent          → Interactive REPL
finance-agent --daemon → Headless daemon (for Docker/systemd)
```

### Daemon Behavior

- Poll Gmail every `daemon.poll_interval` (default 5m)
- Process up to `daemon.batch_size` emails per cycle
- Structured JSON logging (zerolog)
- Optional health endpoint at `localhost:9090/health`
- Graceful shutdown on SIGTERM/SIGINT
- From REPL: `daemon-start` spawns polling as goroutine

### Docker Deployment

```yaml
services:
  finance-agent:
    build: .
    command: ["./finance-agent", "--daemon"]
    env_file: .env
    volumes:
      - ./credentials.json:/app/credentials.json
      - ./token.json:/app/token.json
    depends_on:
      - mongodb
    restart: unless-stopped

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db
```

---

## Configuration

Config loaded via Viper from `./config.yaml` or `./configs/default.yaml`.
Env prefix: `FINANCE_AGENT_`.

```yaml
db:
  uri: "mongodb://localhost:27017"
  database: "finance-agent"
  timeout: 10s

gmail:
  credentials_file: "./credentials.json"
  token_file: "./token.json"
  user: "me"
  labels: ["INBOX"]
  sender_filters:
    - "alerts@hdfcbank.net"
    - "creditcards@hdfcbank.net"

daemon:
  poll_interval: 5m
  batch_size: 50
  health_port: 9090

ai:
  provider: "nvidia"
  api_key_env: "NVIDIA_API_KEY"
  model: "meta/llama-3.1-70b-instruct"
  max_batch: 10

cli:
  theme: "default"
  currency_symbol: "₹"
  date_format: "02 Jan 2006"
  confirm_prompts: true
  verbose: false

parsers:
  default_bank: "hdfc"
  statement_upload_dir: "./statements"

categories:
  auto_learn: true
  min_confidence: 0.8
```

---

## Implementation Phases

| Phase | Scope | Working Commands / Deliverable |
|-------|-------|-------------------------------|
| **P0** | Skeleton — REPL, config, MongoDB, themed output | `start`, `config`, `help`, `exit` |
| **P1** | Accounts — CRUD, balance display, overview | `accounts`, `account-add`, `balance`, `overview` |
| **P2** | Gmail Sync — OAuth, fetch, HDFC parser, store | `sync`, `sync-status`, `txns` |
| **P3** | Categorization — rules, seeds, manual, categories | `categories`, `category-add`, `txn-categorize`, `spend` |
| **P4** | Credit Cards — model, bill parsing, dues | `cards`, `card-add`, `card-bill`, `card-due` |
| **P5** | AI + Polish — NVIDIA fallback, learning, trends | `recategorize`, `spend-trend`, `monthly` |
| **P6** | Daemon — background polling, Docker, import | `daemon-start`, `import` |
| **P7** | Embedded UI — built-in web dashboard | `localhost:3000` dashboard |
| **P8** | iData-UI Integration — REST API for mobile app (future) | — |

---

## Prerequisites

### Gmail API Setup

1. Create Google Cloud project
2. Enable Gmail API
3. Create OAuth 2.0 credentials (desktop app type)
4. Download `credentials.json`
5. First run: browser-based consent → stores `token.json`

### HDFC Email Alerts

Ensure these are enabled in HDFC NetBanking:
- Transaction alerts (debit/credit) to your Gmail
- Credit card transaction alerts
- Credit card statement alerts

### MongoDB

Local: `mongod` running on default port
Production: Atlas connection string in config

### NVIDIA AI

- API key from NVIDIA AI Endpoints or build.nvidia.com
- Set as `NVIDIA_API_KEY` env variable

---

## Embedded UI (Phase 7)

### Overview

A lightweight web dashboard built into the finance-agent binary itself. Provides visual access to all financial data without needing the full iData-UI mobile app. The Go agent serves both the API and the static frontend.

### Tech Stack

| Layer | Choice | Reason |
|-------|--------|--------|
| Frontend | Vite + React + TypeScript | Fast DX, strong typing, huge ecosystem |
| Styling | Tailwind CSS | Utility-first, no custom CSS maintenance |
| Charts | Recharts or Chart.js | Category breakdown, spend trends |
| HTTP client | fetch / TanStack Query | Simple data fetching with caching |
| Build output | Static files (`dist/`) | Embeddable into Go binary |

### Serving Strategy

- **Development**: Vite dev server on `localhost:5173` with proxy to Go API on `:8080`
- **Production**: Go binary embeds `web/dist/` via `embed.FS`, serves at `localhost:3000`
- Single command: `finance-agent --serve` starts both API + UI

### Project Structure Addition

```
finance-agent/
├── ...existing...
├── web/                            # Embedded UI (Vite + React)
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tailwind.config.ts
│   ├── index.html
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx       # Overview — CRED home style
│   │   │   ├── Accounts.tsx        # Account list + balances
│   │   │   ├── Transactions.tsx    # Transaction table + filters
│   │   │   ├── Cards.tsx           # Credit card status
│   │   │   └── Analytics.tsx       # Spend charts + trends
│   │   ├── components/
│   │   │   ├── AccountCard.tsx
│   │   │   ├── TransactionRow.tsx
│   │   │   ├── SpendChart.tsx
│   │   │   ├── CategoryBreakdown.tsx
│   │   │   └── CreditCardWidget.tsx
│   │   ├── hooks/
│   │   │   └── useApi.ts           # TanStack Query hooks
│   │   ├── lib/
│   │   │   └── api.ts              # API client
│   │   └── styles/
│   │       └── globals.css         # Tailwind imports
│   └── dist/                       # Build output (gitignored)
│
├── pkg/api/                        # REST API for UI
│   ├── server.go                   # Gin/Chi router + embed handler
│   ├── handlers_accounts.go
│   ├── handlers_transactions.go
│   ├── handlers_cards.go
│   ├── handlers_analytics.go
│   └── middleware.go               # CORS, logging
```

### API Endpoints (served by Go)

```
GET  /api/overview              — accounts + cards + monthly summary
GET  /api/accounts              — list all accounts
GET  /api/accounts/:id/balance  — single account balance history

GET  /api/transactions          — paginated, filterable transaction list
     ?from=2026-06-01&to=2026-06-30&category=food&search=swiggy&page=1&limit=50
GET  /api/transactions/stats    — spend by category, income vs expense

GET  /api/cards                 — all credit cards
GET  /api/cards/:id/spend       — card spend breakdown

GET  /api/analytics/spend       — category breakdown (current month)
GET  /api/analytics/trend       — month-over-month spend trend
GET  /api/analytics/recurring   — recurring transactions

GET  /api/sync/status           — last sync info
POST /api/sync/trigger          — trigger manual sync from UI
```

### UI Pages

**Dashboard (home)**:
- Total balance card (all accounts summed)
- Account cards row (each bank account with balance)
- Credit card widget (outstanding, due date, utilization bar)
- This month: income vs spend donut
- Recent transactions (last 5)

**Accounts**:
- Account list with expandable details
- Balance trend chart per account
- Add/edit account forms

**Transactions**:
- Searchable, filterable table
- Date range picker, category filter, amount range
- Inline category edit (click to recategorize)
- Pagination

**Cards**:
- Card detail view with limit/outstanding/utilization
- Spend breakdown for current cycle
- Bill history and due date countdown

**Analytics**:
- Category pie/bar chart
- Monthly trend line chart (last 6 months)
- Top merchants list
- Recurring payments detected

### Go Embedding

```go
//go:embed web/dist/*
var webFS embed.FS

func ServeUI(router *gin.Engine) {
    static, _ := fs.Sub(webFS, "web/dist")
    router.NoRoute(gin.WrapH(http.FileServer(http.FS(static))))
}
```

### Build Integration (Makefile)

```makefile
build-web:
    cd web && npm run build

build: build-web
    go build -o bin/finance-agent main.go

dev-web:
    cd web && npm run dev

dev-api:
    go run main.go --serve --dev  # API only, CORS open for vite proxy
```

---

## Future Considerations (Post v0.1)

- **v0.2**: Investments (MF via CAMS/Karvy emails, FD from bank emails)
- **v0.3**: Multi-bank support (ICICI, SBI parsers)
- **v0.4**: Telegram bot for daily reports
- **v0.5**: iData-UI REST API layer
- Budget/goal setting
- Recurring payment detection and alerts
- Net worth tracking over time
- PDF statement OCR for older statements
