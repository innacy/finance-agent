"""Detect subscription confirmation and renewal emails."""

import re
import logging
from datetime import datetime
from decimal import Decimal
from typing import Optional

logger = logging.getLogger(__name__)

SUBSCRIPTION_PATTERNS = {
    "netflix": {"name": "Netflix", "category": "entertainment"},
    "spotify": {"name": "Spotify", "category": "entertainment"},
    "youtube": {"name": "YouTube Premium", "category": "entertainment"},
    "amazon prime": {"name": "Amazon Prime", "category": "entertainment"},
    "hotstar": {"name": "Disney+ Hotstar", "category": "entertainment"},
    "jiocinema": {"name": "JioCinema", "category": "entertainment"},
    "notion": {"name": "Notion", "category": "productivity"},
    "github": {"name": "GitHub", "category": "productivity"},
    "google one": {"name": "Google One", "category": "cloud"},
    "icloud": {"name": "iCloud", "category": "cloud"},
    "dropbox": {"name": "Dropbox", "category": "cloud"},
    "microsoft 365": {"name": "Microsoft 365", "category": "productivity"},
    "adobe": {"name": "Adobe", "category": "productivity"},
    "figma": {"name": "Figma", "category": "productivity"},
    "linkedin premium": {"name": "LinkedIn Premium", "category": "professional"},
    "swiggy one": {"name": "Swiggy One", "category": "food"},
    "zomato gold": {"name": "Zomato Gold", "category": "food"},
    "zomato pro": {"name": "Zomato Pro", "category": "food"},
    "chatgpt": {"name": "ChatGPT Plus", "category": "productivity"},
    "openai": {"name": "OpenAI", "category": "productivity"},
    "claude": {"name": "Claude Pro", "category": "productivity"},
    "cursor": {"name": "Cursor", "category": "productivity"},
    "aws": {"name": "AWS", "category": "cloud"},
    "digitalocean": {"name": "DigitalOcean", "category": "cloud"},
    "vercel": {"name": "Vercel", "category": "cloud"},
}


def parse_subscription_email(email_data: dict) -> Optional[dict]:
    """Detect if email is a subscription confirmation/renewal."""
    subject = email_data.get("subject", "")
    body = email_data.get("body_text") or ""
    sender = email_data.get("from", "")
    date = email_data.get("date", datetime.utcnow())

    combined = f"{subject} {body} {sender}".lower()

    if not any(kw in combined for kw in [
        "subscription", "renewal", "billing", "invoice", "receipt",
        "payment", "plan", "membership", "auto-renew",
    ]):
        return None

    sub_name, sub_info = _detect_subscription(combined)
    if not sub_name:
        return None

    amount = _extract_amount(f"{subject} {body}")
    frequency = _detect_frequency(combined)

    return {
        "name": sub_info.get("name", sub_name),
        "amount": amount,
        "frequency": frequency,
        "category": sub_info.get("category", "other"),
        "date": date,
        "email_id": email_data.get("id", ""),
        "source": "email",
        "auto_detected": True,
    }


def _detect_subscription(text: str) -> tuple[str, dict]:
    for key, info in SUBSCRIPTION_PATTERNS.items():
        if key in text:
            return key, info
    return "", {}


def _extract_amount(text: str) -> Optional[Decimal]:
    patterns = [
        r"(?:Rs\.?|INR|₹|USD|\$)\s*([\d,]+\.?\d*)",
        r"([\d,]+\.?\d*)\s*(?:Rs\.?|INR|₹)",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            try:
                return Decimal(m.group(1).replace(",", ""))
            except Exception:
                continue
    return None


def _detect_frequency(text: str) -> str:
    if any(w in text for w in ["yearly", "annual", "per year", "/year", "per annum"]):
        return "yearly"
    elif any(w in text for w in ["quarterly", "per quarter"]):
        return "quarterly"
    elif any(w in text for w in ["weekly", "per week"]):
        return "weekly"
    return "monthly"
