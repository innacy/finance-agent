"""Extract and normalize transactions from parsed statement data.

Bridges the gap between raw StatementParser output and the Transaction model.
"""

import logging
from datetime import datetime
from decimal import Decimal

from src.db.mongo import compute_dedup_hash

logger = logging.getLogger(__name__)


def normalize_statement_transactions(raw_txns: list[dict],
                                      bank_name: str = "HDFC",
                                      account_masked: str = "") -> list[dict]:
    """Convert raw statement parser output into transaction-ready dicts."""
    normalized = []
    for raw in raw_txns:
        try:
            txn = {
                "date": raw["date"],
                "amount": raw["amount"],
                "type": raw["type"],
                "bank_name": bank_name,
                "account_number_masked": account_masked or raw.get("account_number_masked", ""),
                "description": raw.get("description", ""),
                "payment_mode": _detect_mode_from_description(raw.get("description", "")),
                "merchant": _extract_merchant(raw.get("description", "")),
                "source": "statement",
                "balance": raw.get("balance"),
            }
            txn["dedup_hash"] = compute_dedup_hash(
                txn["date"], txn["amount"],
                txn["description"], txn["account_number_masked"],
            )
            normalized.append(txn)
        except Exception:
            logger.debug("Skipping malformed statement row: %s", raw)
    return normalized


def _detect_mode_from_description(desc: str) -> str:
    d = desc.lower()
    if "upi" in d:
        return "UPI"
    elif "neft" in d:
        return "NEFT"
    elif "imps" in d:
        return "IMPS"
    elif "rtgs" in d:
        return "RTGS"
    elif any(k in d for k in ["atm", "cash withdrawal"]):
        return "cash"
    elif any(k in d for k in ["pos", "ecom", "card"]):
        return "card"
    elif any(k in d for k in ["ach", "nach", "auto"]):
        return "auto_debit"
    elif "chq" in d or "cheque" in d:
        return "cheque"
    return "other"


def _extract_merchant(desc: str) -> str:
    """Try to pull a merchant name from statement narration."""
    import re

    # UPI pattern: UPI-MERCHANT-upiid@bank-...
    m = re.search(r"UPI[/-]([A-Za-z0-9\s&'.]+?)[-/]", desc, re.IGNORECASE)
    if m:
        return m.group(1).strip()

    # NEFT/IMPS pattern: NEFT-...-BENEFICIARY NAME
    m = re.search(r"(?:NEFT|IMPS)[/-]\w+[/-]([A-Za-z\s]+)", desc, re.IGNORECASE)
    if m:
        return m.group(1).strip()

    # POS/card: POS XXXXXXX MERCHANT CITY
    m = re.search(r"POS\s+\d+\s+([A-Za-z\s&'.]+)", desc, re.IGNORECASE)
    if m:
        return m.group(1).strip()

    return ""
