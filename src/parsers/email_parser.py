"""Main email parser -- routes emails to bank-specific or generic parsers
and produces normalized transaction dicts ready for categorization."""

import hashlib
import logging
import re
from datetime import datetime
from decimal import Decimal
from typing import Optional

from src.parsers.templates.hdfc import parse_hdfc_alert
from src.parsers.templates.common import parse_generic_bank_alert

logger = logging.getLogger(__name__)

BANK_PARSERS = {
    "hdfcbank": parse_hdfc_alert,
}


def parse_email(email_data: dict) -> Optional[dict]:
    """Parse a single email into a normalized transaction dict.

    Returns None if the email isn't a parseable financial transaction.
    """
    sender = email_data.get("from", "").lower()
    subject = email_data.get("subject", "")
    body = email_data.get("body_text") or ""
    if not body:
        body = _strip_html(email_data.get("body_html", ""))
    date = email_data.get("date", datetime.utcnow())

    parser = None
    for domain_key, parser_fn in BANK_PARSERS.items():
        if domain_key in sender:
            parser = parser_fn
            break

    if parser:
        result = parser(subject, body, date)
    else:
        result = parse_generic_bank_alert(subject, body, date)

    if not result:
        return None

    result["email_id"] = email_data.get("id", "")
    result["email_subject"] = subject

    if "dedup_hash" not in result or not result["dedup_hash"]:
        result["dedup_hash"] = _compute_dedup_hash(
            result.get("date", date),
            result.get("amount", Decimal("0")),
            result.get("description", ""),
            result.get("account_number_masked", ""),
        )

    return result


def parse_emails_batch(emails: list[dict]) -> list[dict]:
    """Parse a batch of emails, returning only successfully parsed transactions."""
    results = []
    for email_data in emails:
        try:
            txn = parse_email(email_data)
            if txn:
                results.append(txn)
        except Exception:
            logger.exception("Failed to parse email: %s", email_data.get("id", "unknown"))
    logger.info("Parsed %d transactions from %d emails", len(results), len(emails))
    return results


def _strip_html(html: str) -> str:
    text = re.sub(r"<br\s*/?>", "\n", html, flags=re.IGNORECASE)
    text = re.sub(r"<[^>]+>", " ", text)
    text = re.sub(r"&nbsp;", " ", text)
    text = re.sub(r"&amp;", "&", text)
    text = re.sub(r"&lt;", "<", text)
    text = re.sub(r"&gt;", ">", text)
    text = re.sub(r"&#?\w+;", "", text)
    text = re.sub(r"\s+", " ", text).strip()
    return text


def _compute_dedup_hash(date: datetime, amount: Decimal, description: str,
                        account_masked: str) -> str:
    raw = f"{date.isoformat()}|{amount}|{description.strip().lower()[:100]}|{account_masked}"
    return hashlib.sha256(raw.encode()).hexdigest()[:32]
