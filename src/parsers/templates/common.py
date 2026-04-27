"""Generic Indian bank email parsing -- fallback for banks without specific templates."""

import re
from datetime import datetime
from decimal import Decimal
from typing import Optional


def parse_generic_bank_alert(subject: str, body: str, date: datetime) -> Optional[dict]:
    """Attempt to parse any Indian bank transaction email using common patterns."""
    body_clean = re.sub(r"<[^>]+>", " ", body)
    body_clean = re.sub(r"\s+", " ", body_clean).strip()

    amount = _extract_amount(body_clean)
    if not amount:
        return None

    txn_type = _detect_type(body_clean)
    if not txn_type:
        return None

    return {
        "date": date,
        "amount": amount,
        "type": txn_type,
        "bank_name": _detect_bank(subject, body_clean),
        "account_number_masked": _extract_account(body_clean),
        "payment_mode": _detect_payment_mode(body_clean),
        "merchant": _extract_merchant(body_clean),
        "description": body_clean[:500],
        "source": "email",
    }


def _extract_amount(text: str) -> Optional[Decimal]:
    patterns = [
        r"(?:Rs\.?|INR|₹)\s*([\d,]+\.?\d*)",
        r"([\d,]+\.?\d*)\s*(?:Rs\.?|INR|₹)",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            val = m.group(1).replace(",", "")
            try:
                return Decimal(val)
            except Exception:
                continue
    return None


def _detect_type(text: str) -> Optional[str]:
    lower = text.lower()
    if any(w in lower for w in ["debited", "debit", "withdrawn", "spent", "paid", "purchase"]):
        return "debit"
    elif any(w in lower for w in ["credited", "credit", "received", "deposited", "refund"]):
        return "credit"
    return None


def _extract_account(text: str) -> str:
    m = re.search(r"(?:XX|xx|\*{2,})(\d{4})", text)
    return f"XX{m.group(1)}" if m else ""


def _detect_payment_mode(text: str) -> str:
    lower = text.lower()
    if "upi" in lower:
        return "UPI"
    elif "neft" in lower:
        return "NEFT"
    elif "imps" in lower:
        return "IMPS"
    elif any(k in lower for k in ["card", "pos"]):
        return "card"
    elif "auto" in lower and "debit" in lower:
        return "auto_debit"
    return "other"


def _extract_merchant(text: str) -> str:
    patterns = [
        r"(?:to|at|towards)\s+([A-Za-z0-9\s&'.,]+?)(?:\s+(?:on|via|ref)|[.\n]|$)",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            merchant = m.group(1).strip().rstrip(".")
            if len(merchant) > 2:
                return merchant[:100]
    return ""


def _detect_bank(subject: str, body: str) -> str:
    combined = (subject + " " + body).lower()
    bank_map = {
        "hdfc": "HDFC", "sbi": "SBI", "icici": "ICICI",
        "axis": "AXIS", "kotak": "KOTAK", "bob": "BOB",
        "pnb": "PNB", "canara": "CANARA", "union": "UNION",
        "idbi": "IDBI", "yes bank": "YES", "indusind": "INDUSIND",
    }
    for key, name in bank_map.items():
        if key in combined:
            return name
    return "UNKNOWN"
