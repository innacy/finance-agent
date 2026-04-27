"""SMS data reader -- only applicable when agent runs on mobile (Termux/Android).

Reads SMS from Android's content provider via Termux API or accepts
pre-exported SMS JSON/CSV files.
"""

import json
import logging
import subprocess
from datetime import datetime
from typing import Optional

logger = logging.getLogger(__name__)


def is_mobile_environment() -> bool:
    """Check if running in Termux / Android environment."""
    try:
        result = subprocess.run(["termux-info"], capture_output=True, timeout=5)
        return result.returncode == 0
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return False


def read_sms_termux(limit: int = 100, offset: int = 0) -> list[dict]:
    """Read SMS messages using Termux API (Android only)."""
    if not is_mobile_environment():
        logger.debug("Not a mobile environment, skipping SMS read")
        return []

    try:
        result = subprocess.run(
            ["termux-sms-list", "-l", str(limit), "-o", str(offset)],
            capture_output=True, text=True, timeout=30,
        )
        if result.returncode != 0:
            logger.warning("termux-sms-list failed: %s", result.stderr)
            return []
        return json.loads(result.stdout)
    except Exception:
        logger.exception("Failed to read SMS via Termux")
        return []


def read_sms_from_file(file_path: str) -> list[dict]:
    """Read pre-exported SMS JSON file."""
    try:
        with open(file_path, "r") as f:
            data = json.load(f)
        if isinstance(data, list):
            return data
        return data.get("messages", data.get("sms", []))
    except Exception:
        logger.exception("Failed to read SMS file: %s", file_path)
        return []


def filter_financial_sms(messages: list[dict]) -> list[dict]:
    """Filter SMS messages that are likely financial transactions."""
    financial_keywords = [
        "debited", "credited", "withdrawn", "deposited", "txn",
        "transaction", "a/c", "acct", "upi", "neft", "imps",
        "emi", "loan", "insurance", "premium", "balance",
        "credit card", "debit card",
    ]
    financial = []
    for msg in messages:
        body = (msg.get("body") or msg.get("message") or "").lower()
        if any(kw in body for kw in financial_keywords):
            financial.append(msg)
    return financial
