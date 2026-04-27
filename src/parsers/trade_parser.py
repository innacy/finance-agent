"""Parse trade confirmation emails from Zerodha and Groww."""

import re
import logging
from datetime import datetime
from decimal import Decimal
from typing import Optional

logger = logging.getLogger(__name__)


def parse_trade_email(email_data: dict) -> Optional[dict]:
    sender = email_data.get("from", "").lower()
    subject = email_data.get("subject", "")
    body = email_data.get("body_text") or ""
    date = email_data.get("date", datetime.utcnow())

    if "zerodha" in sender:
        return _parse_zerodha(subject, body, date, email_data)
    elif "groww" in sender:
        return _parse_groww(subject, body, date, email_data)
    return None


def _parse_zerodha(subject: str, body: str, date: datetime,
                    email_data: dict) -> Optional[dict]:
    if "contract note" not in subject.lower():
        return None

    trades = []
    # Zerodha contract notes are typically PDF attachments
    if email_data.get("attachments"):
        return {
            "type": "contract_note",
            "broker": "zerodha",
            "date": date,
            "has_attachment": True,
            "attachments": email_data["attachments"],
            "source": "email",
        }

    lines = body.split("\n")
    for line in lines:
        trade = _extract_trade_line(line, "zerodha")
        if trade:
            trade["date"] = date
            trades.append(trade)

    if trades:
        return {
            "type": "trades",
            "broker": "zerodha",
            "date": date,
            "trades": trades,
            "source": "email",
        }
    return None


def _parse_groww(subject: str, body: str, date: datetime,
                  email_data: dict) -> Optional[dict]:
    subject_lower = subject.lower()
    if "order executed" not in subject_lower and "order placed" not in subject_lower:
        return None

    trade_type = "buy" if "buy" in body.lower() else "sell" if "sell" in body.lower() else None

    amount = _extract_amount(body)
    name = _extract_security_name(body)
    units = _extract_units(body)

    if amount:
        return {
            "type": "trade",
            "broker": "groww",
            "date": date,
            "trade_type": trade_type,
            "security_name": name,
            "amount": amount,
            "units": units,
            "source": "email",
        }
    return None


def _extract_trade_line(line: str, broker: str) -> Optional[dict]:
    m = re.search(
        r"(BUY|SELL)\s+(\d+\.?\d*)\s+(?:shares?\s+of\s+)?([A-Z0-9]+)\s+"
        r"(?:@|at)\s*(?:Rs\.?|₹|INR)?\s*([\d,.]+)",
        line, re.IGNORECASE
    )
    if m:
        return {
            "trade_type": m.group(1).lower(),
            "units": Decimal(m.group(2)),
            "symbol": m.group(3),
            "price": Decimal(m.group(4).replace(",", "")),
            "broker": broker,
        }
    return None


def _extract_amount(text: str) -> Optional[Decimal]:
    m = re.search(r"(?:Rs\.?|INR|₹)\s*([\d,]+\.?\d*)", text, re.IGNORECASE)
    if m:
        try:
            return Decimal(m.group(1).replace(",", ""))
        except Exception:
            pass
    return None


def _extract_security_name(text: str) -> str:
    patterns = [
        r"(?:of|in)\s+([A-Z][A-Za-z0-9\s&.]+?)(?:\s+(?:at|@|for|has))",
        r"Stock:\s*([^\n]+)",
        r"Fund:\s*([^\n]+)",
    ]
    for p in patterns:
        m = re.search(p, text)
        if m:
            return m.group(1).strip()
    return ""


def _extract_units(text: str) -> Optional[Decimal]:
    patterns = [
        r"(\d+\.?\d*)\s*(?:shares?|units?|qty)",
        r"(?:quantity|qty)\s*:?\s*(\d+\.?\d*)",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            try:
                return Decimal(m.group(1))
            except Exception:
                pass
    return None
