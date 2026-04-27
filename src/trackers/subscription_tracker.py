"""Track and auto-detect subscriptions from recurring transactions."""

import logging
from collections import defaultdict
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from src.db.mongo import (
    subscriptions_col, transactions_col,
    _restore_decimals, _convert_decimals,
)

logger = logging.getLogger(__name__)

FREQUENCY_DAYS = {
    "weekly": 7,
    "monthly": 30,
    "quarterly": 90,
    "yearly": 365,
}


def add_subscription(user_id: str, name: str, amount: Decimal,
                      frequency: str = "monthly", category: str = "",
                      payment_method: str = "",
                      next_billing_date: Optional[datetime] = None) -> str:
    doc = _convert_decimals({
        "user_id": user_id,
        "name": name,
        "amount": amount,
        "frequency": frequency,
        "next_billing_date": next_billing_date,
        "category": category,
        "payment_method": payment_method,
        "auto_detected": False,
        "status": "active",
        "created_at": datetime.utcnow(),
    })
    result = subscriptions_col().insert_one(doc)
    return str(result.inserted_id)


def detect_recurring_transactions(user_id: str = "default",
                                   lookback_days: int = 90,
                                   min_occurrences: int = 2) -> list[dict]:
    """Auto-detect subscriptions by finding recurring same-amount transactions
    to the same merchant."""
    start = datetime.utcnow() - timedelta(days=lookback_days)

    pipeline = [
        {"$match": {
            "user_id": user_id,
            "type": "debit",
            "date": {"$gte": start},
            "merchant": {"$ne": ""},
        }},
        {"$group": {
            "_id": {"merchant": "$merchant", "amount": "$amount"},
            "count": {"$sum": 1},
            "dates": {"$push": "$date"},
            "category": {"$first": "$category"},
        }},
        {"$match": {"count": {"$gte": min_occurrences}}},
        {"$sort": {"count": -1}},
    ]

    results = list(transactions_col().aggregate(pipeline))
    detected = []

    for item in results:
        merchant = item["_id"]["merchant"]
        amount = item["_id"]["amount"]
        dates = sorted(item["dates"])
        frequency = _detect_frequency(dates)

        if frequency:
            existing = subscriptions_col().find_one({
                "user_id": user_id,
                "name": {"$regex": merchant, "$options": "i"},
            })
            if existing:
                continue

            sub = {
                "name": merchant,
                "amount": Decimal(str(amount)) if not isinstance(amount, Decimal) else amount,
                "frequency": frequency,
                "category": item.get("category", "subscriptions"),
                "occurrences": item["count"],
                "last_date": dates[-1],
                "next_billing_date": _estimate_next_billing(dates[-1], frequency),
            }
            detected.append(sub)

    return detected


def auto_register_subscriptions(user_id: str = "default") -> int:
    """Detect and register new subscriptions."""
    detected = detect_recurring_transactions(user_id)
    count = 0
    for sub in detected:
        doc = _convert_decimals({
            "user_id": user_id,
            "name": sub["name"],
            "amount": sub["amount"],
            "frequency": sub["frequency"],
            "next_billing_date": sub.get("next_billing_date"),
            "category": sub.get("category", "subscriptions"),
            "payment_method": "",
            "auto_detected": True,
            "status": "active",
            "created_at": datetime.utcnow(),
        })
        subscriptions_col().update_one(
            {"user_id": user_id, "name": sub["name"]},
            {"$setOnInsert": doc},
            upsert=True,
        )
        count += 1
    logger.info("Auto-detected %d new subscriptions", count)
    return count


def get_monthly_subscription_cost(user_id: str = "default") -> Decimal:
    total = Decimal("0")
    for sub in subscriptions_col().find({"user_id": user_id, "status": "active"}):
        sub = _restore_decimals(sub)
        amount = sub.get("amount", Decimal("0"))
        freq = sub.get("frequency", "monthly")
        if freq == "yearly":
            total += amount / 12
        elif freq == "quarterly":
            total += amount / 3
        elif freq == "weekly":
            total += amount * Decimal("4.33")
        else:
            total += amount
    return total


def _detect_frequency(dates: list[datetime]) -> Optional[str]:
    if len(dates) < 2:
        return None

    gaps = [(dates[i+1] - dates[i]).days for i in range(len(dates)-1)]
    avg_gap = sum(gaps) / len(gaps)

    if 5 <= avg_gap <= 10:
        return "weekly"
    elif 25 <= avg_gap <= 35:
        return "monthly"
    elif 80 <= avg_gap <= 100:
        return "quarterly"
    elif 340 <= avg_gap <= 390:
        return "yearly"
    return None


def _estimate_next_billing(last_date: datetime, frequency: str) -> datetime:
    days = FREQUENCY_DAYS.get(frequency, 30)
    return last_date + timedelta(days=days)
