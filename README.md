# Finance Agent

Personal finance tracking agent (CRED Money style). Syncs bank transactions from Gmail, tracks investments/loans/subscriptions, categorizes spending intelligently, stores in MongoDB, and sends Telegram reports twice daily.

## Features

- **Bank Transaction Sync** -- Gmail API reads HDFC (and other bank) alerts automatically
- **Statement Import** -- Upload PDF/CSV bank statements for historical data
- **Smart Categorization** -- Rule-based engine (500+ Indian merchants) + Gemini AI fallback
- **Investment Tracking** -- Stocks & mutual funds via CAS statement + email parsing
- **Loan & EMI Tracking** -- Home/car/personal loans with EMI schedule
- **Credit Card Tracking** -- Bills, dues, spending breakdown
- **Insurance Tracking** -- Premium due dates and coverage
- **FD/RD Tracking** -- Maturity dates and current value
- **Subscription Detection** -- Auto-detects recurring payments
- **Net Worth Calculator** -- Daily snapshots across all accounts
- **Spending Analytics** -- Category trends, month-over-month, unusual alerts
- **Telegram Bot** -- Morning/evening reports + on-demand commands
- **REST API** -- For future iData-ui dashboard integration

## Quick Start

### 1. Clone and configure

```bash
cd finance-agent
cp .env.example .env
# Edit .env with your credentials
```

### 2. Set up Gmail OAuth

```bash
# 1. Create OAuth credentials at https://console.cloud.google.com/apis/credentials
# 2. Download JSON → save as data/gmail_credentials.json
# 3. Run:
pip install -r requirements.txt
python -m scripts.setup_gmail_oauth
```

### 3. Set up Telegram Bot

```bash
python -m scripts.setup_telegram
# Follow the prompts (create bot via @BotFather)
```

### 4. Run with Docker

```bash
docker compose up -d
```

Or run locally:

```bash
python -m src.main
```

### 5. Import historical data

```bash
# Bank statements
python -m scripts.initial_import --type statement --file data/hdfc_2024.pdf --bank hdfc --account XX1234

# Mutual fund CAS
python -m scripts.initial_import --type cas --file data/cas_statement.pdf

# All statements from a directory
python -m scripts.initial_import --type statement --dir data/statements/ --bank hdfc --account XX1234
```

## Telegram Commands


| Command          | Description                          |
| ---------------- | ------------------------------------ |
| `/balance`       | All account balances                 |
| `/spend [days]`  | Spending breakdown (default: 7 days) |
| `/networth`      | Current net worth                    |
| `/upcoming`      | Upcoming EMIs, premiums, bills       |
| `/investments`   | Portfolio summary                    |
| `/subscriptions` | Active subscriptions                 |
| `/morning`       | Full morning report                  |
| `/evening`       | Full evening report                  |
| `/sync`          | Trigger manual sync                  |


## API Endpoints


| Method | Endpoint                                            | Description            |
| ------ | --------------------------------------------------- | ---------------------- |
| GET    | `/api/v1/accounts`                                  | All bank accounts      |
| GET    | `/api/v1/transactions?days=30&category=food_dining` | Transaction history    |
| GET    | `/api/v1/spending?days=30`                          | Spending by category   |
| GET    | `/api/v1/spending/trend?months=6`                   | Monthly spending trend |
| GET    | `/api/v1/spending/insights`                         | AI spending insights   |
| GET    | `/api/v1/spending/merchants?days=30`                | Top merchants          |
| GET    | `/api/v1/spending/unusual`                          | Unusual transactions   |
| GET    | `/api/v1/networth`                                  | Latest net worth       |
| GET    | `/api/v1/networth/history?days=30`                  | Net worth history      |
| GET    | `/api/v1/investments`                               | Portfolio holdings     |
| GET    | `/api/v1/subscriptions`                             | Active subscriptions   |
| GET    | `/api/v1/loans`                                     | Loan details           |
| GET    | `/api/v1/insurance`                                 | Insurance policies     |
| GET    | `/api/v1/fd-rd`                                     | FD/RD details          |
| POST   | `/api/v1/sync`                                      | Trigger manual sync    |
| POST   | `/api/v1/statements/upload`                         | Upload bank statement  |
| POST   | `/api/v1/statements/upload-cas`                     | Upload CAS PDF         |


## Environment Variables


| Variable                 | Description                  | Default                          |
| ------------------------ | ---------------------------- | -------------------------------- |
| `MONGO_URI`              | MongoDB connection string    | `mongodb://localhost:27017`      |
| `DATABASE_NAME`          | MongoDB database name        | `finance_agent`                  |
| `GMAIL_CREDENTIALS_PATH` | Gmail OAuth credentials JSON | `./data/gmail_credentials.json`  |
| `GEMINI_API_KEY`         | Google Gemini API key        | (required for AI categorization) |
| `TELEGRAM_BOT_TOKEN`     | Telegram bot token           | (required for reports)           |
| `TELEGRAM_CHAT_ID`       | Your Telegram chat ID        | (required for reports)           |
| `SCHEDULE_MORNING`       | Morning sync time (IST)      | `07:00`                          |
| `SCHEDULE_EVENING`       | Evening sync time (IST)      | `19:00`                          |
| `TIMEZONE`               | Timezone                     | `Asia/Kolkata`                   |


## Testing

```bash
pytest tests/ -v
```

