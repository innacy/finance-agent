"""Track insurance policies and premium due dates."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from src.db.mongo import insurance_col, _restore_decimals, _convert_decimals

logger = logging.getLogger(__name__)

FREQUENCY_MONTHS = {
    "monthly": 1,
    "quarterly": 3,
    "half_yearly": 6,
    "yearly": 12,
}


def add_policy(user_id: str, ins_type: str, provider: str,
               premium_amount: Decimal, frequency: str = "yearly",
               sum_assured: Decimal = Decimal("0"),
               policy_number_masked: str = "",
               next_premium_date: Optional[datetime] = None) -> str:
    doc = _convert_decimals({
        "user_id": user_id,
        "type": ins_type,
        "provider": provider,
        "policy_number_masked": policy_number_masked,
        "premium_amount": premium_amount,
        "premium_frequency": frequency,
        "next_premium_date": next_premium_date,
        "sum_assured": sum_assured,
        "status": "active",
        "created_at": datetime.utcnow(),
    })
    result = insurance_col().insert_one(doc)
    return str(result.inserted_id)


def record_premium_payment(user_id: str, provider: str):
    """Mark a premium as paid and advance the next due date."""
    policy = insurance_col().find_one(
        {"user_id": user_id, "provider": provider, "status": "active"}
    )
    if not policy:
        return

    policy = _restore_decimals(policy)
    freq = policy.get("premium_frequency", "yearly")
    months = FREQUENCY_MONTHS.get(freq, 12)

    current_due = policy.get("next_premium_date") or datetime.utcnow()
    next_due = _advance_date(current_due, months)

    insurance_col().update_one(
        {"_id": policy["_id"]},
        {"$set": {"next_premium_date": next_due}},
    )


def get_upcoming_premiums(user_id: str, days: int = 30) -> list[dict]:
    cutoff = datetime.utcnow() + timedelta(days=days)
    policies = insurance_col().find({
        "user_id": user_id,
        "status": "active",
        "next_premium_date": {"$lte": cutoff, "$gte": datetime.utcnow()},
    })
    return [_restore_decimals(p) for p in policies]


def get_total_annual_premiums(user_id: str = "default") -> Decimal:
    """Calculate total annual insurance premium cost."""
    total = Decimal("0")
    for policy in insurance_col().find({"user_id": user_id, "status": "active"}):
        policy = _restore_decimals(policy)
        premium = policy.get("premium_amount", Decimal("0"))
        freq = policy.get("premium_frequency", "yearly")
        multiplier = 12 // FREQUENCY_MONTHS.get(freq, 12)
        total += premium * multiplier
    return total


def _advance_date(dt: datetime, months: int) -> datetime:
    month = dt.month + months
    year = dt.year
    while month > 12:
        month -= 12
        year += 1
    return dt.replace(year=year, month=month)
