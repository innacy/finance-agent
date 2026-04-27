"""HDFC Bank email parsing templates.

Covers: debit/credit alerts (UPI, NEFT, IMPS, card), credit card transactions,
FD/RD notifications, loan EMI debits.
"""

import re
from datetime import datetime
from decimal import Decimal
from typing import Optional


def parse_hdfc_alert(subject: str, body: str, date: datetime) -> Optional[dict]:
    """Route to the appropriate HDFC parser based on email content."""
    body_lower = body.lower()

    if "emi" in body_lower:
        return _parse_emi_alert(body, date)
    elif "insurance" in body_lower or "premium" in body_lower:
        return _parse_insurance_alert(body, date)
    elif "credit card" in subject.lower():
        return _parse_cc_alert(body, date)
    elif "fixed deposit" in body_lower:
        return _parse_fd_alert(body, date)
    elif "recurring deposit" in body_lower or "rd " in body_lower:
        return _parse_rd_alert(body, date)
    elif "debited" in body_lower or "withdrawn" in body_lower:
        return _parse_debit_alert(body, date)
    elif "credited" in body_lower:
        return _parse_credit_alert(body, date)
    return None


def _extract_amount(text: str) -> Optional[Decimal]:
    patterns = [
        r"(?:Rs\.?|INR|₹)\s*([\d,]+\.?\d*)",
        r"([\d,]+\.?\d*)\s*(?:Rs\.?|INR|₹)",
        r"amount\s+(?:of\s+)?(?:Rs\.?|INR|₹)?\s*([\d,]+\.?\d*)",
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


def _extract_account(text: str) -> str:
    patterns = [
        r"a/c\s*(?:no\.?\s*)?(?:\*+)?(\w{4,})",
        r"account\s*(?:no\.?\s*)?(?:\*+)?(\w{4,})",
        r"(?:XX|xx|\*{2,})(\d{4})",
        r"ending\s+(\d{4})",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            return f"XX{m.group(1)[-4:]}"
    return ""


def _extract_upi_id(text: str) -> str:
    m = re.search(r"([a-zA-Z0-9._-]+@[a-zA-Z0-9]+)", text)
    return m.group(1) if m else ""


def _extract_ref_number(text: str) -> str:
    patterns = [
        r"ref\s*(?:no\.?\s*)?:?\s*(\d{6,})",
        r"reference\s*(?:no\.?\s*)?:?\s*(\d{6,})",
        r"utr\s*(?:no\.?\s*)?:?\s*(\w{6,})",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            return m.group(1)
    return ""


def _extract_merchant_from_description(text: str) -> str:
    patterns = [
        r"(?:to|at|towards|paid to|transferred to|for)\s+([A-Za-z0-9\s&'.,-]+?)(?:\s+(?:on|via|ref|upi|neft|imps)|$)",
        r"(?:from)\s+([A-Za-z0-9\s&'.,-]+?)(?:\s+(?:on|via|ref|upi|neft|imps)|$)",
        r"VPA\s+\S+\s*\(([^)]+)\)",
    ]
    for p in patterns:
        m = re.search(p, text, re.IGNORECASE)
        if m:
            merchant = m.group(1).strip().rstrip(".")
            if len(merchant) > 2 and not merchant.isdigit():
                return merchant
    return ""


def _detect_payment_mode(text: str) -> str:
    text_lower = text.lower()
    if "upi" in text_lower or "@" in text:
        return "UPI"
    elif "neft" in text_lower:
        return "NEFT"
    elif "imps" in text_lower:
        return "IMPS"
    elif "rtgs" in text_lower:
        return "RTGS"
    elif "atm" in text_lower:
        return "cash"
    elif any(k in text_lower for k in ["card", "pos", "swipe", "ecom"]):
        return "card"
    elif "auto" in text_lower and "debit" in text_lower:
        return "auto_debit"
    elif "cheque" in text_lower or "chq" in text_lower:
        return "cheque"
    return "other"


def _parse_debit_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    if not amount:
        return None

    return {
        "date": date,
        "amount": amount,
        "type": "debit",
        "bank_name": "HDFC",
        "account_number_masked": _extract_account(body),
        "payment_mode": _detect_payment_mode(body),
        "merchant": _extract_merchant_from_description(body),
        "upi_id": _extract_upi_id(body),
        "ref_number": _extract_ref_number(body),
        "description": body.strip()[:500],
        "source": "email",
    }


def _parse_credit_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    if not amount:
        return None

    return {
        "date": date,
        "amount": amount,
        "type": "credit",
        "bank_name": "HDFC",
        "account_number_masked": _extract_account(body),
        "payment_mode": _detect_payment_mode(body),
        "merchant": _extract_merchant_from_description(body),
        "upi_id": _extract_upi_id(body),
        "ref_number": _extract_ref_number(body),
        "description": body.strip()[:500],
        "source": "email",
    }


def _parse_cc_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    if not amount:
        return None

    return {
        "date": date,
        "amount": amount,
        "type": "debit",
        "bank_name": "HDFC",
        "account_number_masked": _extract_account(body),
        "payment_mode": "card",
        "merchant": _extract_merchant_from_description(body),
        "description": body.strip()[:500],
        "source": "email",
        "is_credit_card": True,
    }


def _parse_fd_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    return {
        "date": date,
        "amount": amount,
        "type": "fd_notification",
        "bank_name": "HDFC",
        "description": body.strip()[:500],
        "source": "email",
    }


def _parse_rd_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    return {
        "date": date,
        "amount": amount,
        "type": "rd_notification",
        "bank_name": "HDFC",
        "description": body.strip()[:500],
        "source": "email",
    }


def _parse_emi_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    return {
        "date": date,
        "amount": amount,
        "type": "debit",
        "bank_name": "HDFC",
        "account_number_masked": _extract_account(body),
        "payment_mode": "auto_debit",
        "merchant": "EMI",
        "description": body.strip()[:500],
        "source": "email",
        "is_emi": True,
    }


def _parse_insurance_alert(body: str, date: datetime) -> Optional[dict]:
    amount = _extract_amount(body)
    return {
        "date": date,
        "amount": amount,
        "type": "debit",
        "bank_name": "HDFC",
        "payment_mode": "auto_debit",
        "merchant": "Insurance Premium",
        "description": body.strip()[:500],
        "source": "email",
        "is_insurance": True,
    }
