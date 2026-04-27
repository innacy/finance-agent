"""Bank-specific email sender addresses and subject patterns for filtering."""

BANK_EMAIL_FILTERS: dict[str, dict] = {
    "hdfc": {
        "senders": [
            "alerts@hdfcbank.net",
            "noreply@hdfcbank.net",
            "creditcards@hdfcbank.net",
            "hdfcbank.net",
        ],
        "subject_patterns": [
            "Alert : Update on your HDFC Bank",
            "Transaction alert",
            "Credit Card Transaction",
            "Credit Card Statement",
            "HDFC Bank Credit Card",
        ],
    },
    "sbi": {
        "senders": [
            "donotreply@sbi.co.in",
            "noreply@sbi.co.in",
        ],
        "subject_patterns": [
            "SBI Transaction Alert",
            "SBI Credit Card",
        ],
    },
    "icici": {
        "senders": [
            "customer.care@icicibank.com",
            "alert@icicibank.com",
        ],
        "subject_patterns": [
            "ICICI Bank Transaction",
            "ICICI Credit Card",
        ],
    },
}

BROKER_EMAIL_FILTERS: dict[str, dict] = {
    "zerodha": {
        "senders": ["noreply@zerodha.com", "support@zerodha.com"],
        "subject_patterns": ["Contract note", "Zerodha"],
    },
    "groww": {
        "senders": ["noreply@groww.in", "support@groww.in"],
        "subject_patterns": ["Order executed", "Groww"],
    },
}

SUBSCRIPTION_SENDERS: list[str] = [
    "noreply@netflix.com",
    "noreply@spotify.com",
    "no-reply@accounts.google.com",
    "noreply@youtube.com",
    "receipt@amazon.in",
    "auto-confirm@amazon.in",
    "do-not-reply@swiggy.in",
    "noreply@zomato.com",
    "support@notion.so",
    "billing@stripe.com",
]

INSURANCE_SENDERS: list[str] = [
    "noreply@licindia.in",
    "noreply@hdfclife.com",
    "noreply@iciciprulife.com",
    "noreply@maxlifeinsurance.com",
    "noreply@starhealth.in",
    "noreply@careinsurance.com",
]


def build_gmail_query() -> str:
    """Build a Gmail search query that captures all financial emails."""
    all_senders: list[str] = []
    for bank in BANK_EMAIL_FILTERS.values():
        all_senders.extend(bank["senders"])
    for broker in BROKER_EMAIL_FILTERS.values():
        all_senders.extend(broker["senders"])
    all_senders.extend(SUBSCRIPTION_SENDERS)
    all_senders.extend(INSURANCE_SENDERS)

    from_clause = " OR ".join(f"from:{s}" for s in set(all_senders))
    return f"({from_clause})"
