"""Spending analytics -- category trends, month-over-month, unusual alerts."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from src.db.mongo import transactions_col, _restore_decimals
from src.categorization.categories import CATEGORY_DISPLAY_NAMES
from src.reports.formatters import format_currency

logger = logging.getLogger(__name__)


def monthly_spending_trend(user_id: str = "default",
                            months: int = 6) -> list[dict]:
    """Get total spending per month for the last N months."""
    now = datetime.utcnow()
    results = []

    for i in range(months):
        month_start = (now.replace(day=1) - timedelta(days=30 * i)).replace(day=1)
        if month_start.month == 12:
            month_end = month_start.replace(year=month_start.year + 1, month=1)
        else:
            month_end = month_start.replace(month=month_start.month + 1)

        pipeline = [
            {"$match": {
                "user_id": user_id,
                "type": "debit",
                "date": {"$gte": month_start, "$lt": month_end},
            }},
            {"$group": {
                "_id": None,
                "total": {"$sum": "$amount"},
                "count": {"$sum": 1},
            }},
        ]

        agg = list(transactions_col().aggregate(pipeline))
        total = Decimal(str(agg[0]["total"])) if agg else Decimal("0")
        count = agg[0]["count"] if agg else 0

        results.append({
            "month": month_start.strftime("%b %Y"),
            "month_start": month_start,
            "total_spending": total,
            "transaction_count": count,
        })

    results.reverse()
    return results


def category_trend(user_id: str = "default", category: str = "food_dining",
                    months: int = 6) -> list[dict]:
    """Track spending in a specific category over months."""
    now = datetime.utcnow()
    results = []

    for i in range(months):
        month_start = (now.replace(day=1) - timedelta(days=30 * i)).replace(day=1)
        if month_start.month == 12:
            month_end = month_start.replace(year=month_start.year + 1, month=1)
        else:
            month_end = month_start.replace(month=month_start.month + 1)

        pipeline = [
            {"$match": {
                "user_id": user_id,
                "type": "debit",
                "category": category,
                "date": {"$gte": month_start, "$lt": month_end},
            }},
            {"$group": {
                "_id": None,
                "total": {"$sum": "$amount"},
                "count": {"$sum": 1},
            }},
        ]

        agg = list(transactions_col().aggregate(pipeline))
        total = Decimal(str(agg[0]["total"])) if agg else Decimal("0")

        results.append({
            "month": month_start.strftime("%b %Y"),
            "total": total,
            "count": agg[0]["count"] if agg else 0,
        })

    results.reverse()
    return results


def month_over_month_change(user_id: str = "default") -> dict:
    """Compare this month's spending with last month."""
    now = datetime.utcnow()
    this_month_start = now.replace(day=1, hour=0, minute=0, second=0)
    last_month_end = this_month_start - timedelta(seconds=1)
    last_month_start = last_month_end.replace(day=1, hour=0, minute=0, second=0)

    this_month = _total_spending(user_id, this_month_start, now)
    last_month = _total_spending(user_id, last_month_start, last_month_end)

    change = this_month - last_month
    pct = float(change / last_month * 100) if last_month > 0 else 0

    return {
        "this_month": this_month,
        "last_month": last_month,
        "change": change,
        "percent": pct,
        "direction": "up" if change > 0 else "down" if change < 0 else "flat",
    }


def top_merchants(user_id: str = "default", days: int = 30,
                   limit: int = 10) -> list[dict]:
    """Top merchants by spending amount."""
    start = datetime.utcnow() - timedelta(days=days)
    pipeline = [
        {"$match": {
            "user_id": user_id,
            "type": "debit",
            "date": {"$gte": start},
            "merchant": {"$ne": ""},
        }},
        {"$group": {
            "_id": "$merchant",
            "total": {"$sum": "$amount"},
            "count": {"$sum": 1},
            "category": {"$first": "$category"},
        }},
        {"$sort": {"total": -1}},
        {"$limit": limit},
    ]
    return list(transactions_col().aggregate(pipeline))


def detect_unusual_transactions(user_id: str = "default",
                                  threshold_multiplier: float = 3.0) -> list[dict]:
    """Find transactions significantly above the average for their category."""
    now = datetime.utcnow()
    lookback = now - timedelta(days=90)
    recent = now - timedelta(days=7)

    # Get average by category
    avg_pipeline = [
        {"$match": {
            "user_id": user_id, "type": "debit",
            "date": {"$gte": lookback, "$lt": recent},
        }},
        {"$group": {
            "_id": "$category",
            "avg_amount": {"$avg": "$amount"},
            "std_dev": {"$stdDevPop": "$amount"},
        }},
    ]
    averages = {r["_id"]: r for r in transactions_col().aggregate(avg_pipeline)}

    # Check recent transactions
    unusual = []
    for txn in transactions_col().find({
        "user_id": user_id, "type": "debit",
        "date": {"$gte": recent},
    }):
        txn = _restore_decimals(txn)
        cat = txn.get("category", "others")
        avg = averages.get(cat, {}).get("avg_amount", 0)
        if avg and float(txn.get("amount", 0)) > float(avg) * threshold_multiplier:
            unusual.append({
                "date": txn["date"],
                "amount": txn["amount"],
                "merchant": txn.get("merchant", ""),
                "category": cat,
                "avg_for_category": Decimal(str(avg)),
                "multiplier": float(txn["amount"]) / float(avg),
            })

    return unusual


def spending_insights(user_id: str = "default") -> list[str]:
    """Generate human-readable spending insights."""
    insights = []

    mom = month_over_month_change(user_id)
    if mom["percent"] > 20:
        insights.append(
            f"Spending is up {mom['percent']:.0f}% this month "
            f"({format_currency(mom['this_month'])} vs "
            f"{format_currency(mom['last_month'])} last month)"
        )
    elif mom["percent"] < -20:
        insights.append(
            f"Great job! Spending is down {abs(mom['percent']):.0f}% this month"
        )

    merchants = top_merchants(user_id, days=30, limit=3)
    if merchants:
        top = merchants[0]
        insights.append(
            f"Top spending: {top['_id']} ({format_currency(Decimal(str(top['total'])))})"
        )

    unusual = detect_unusual_transactions(user_id)
    for u in unusual[:2]:
        insights.append(
            f"Unusual: {format_currency(u['amount'])} at {u['merchant']} "
            f"({u['multiplier']:.1f}x your average for {u['category']})"
        )

    return insights


def _total_spending(user_id: str, start: datetime, end: datetime) -> Decimal:
    pipeline = [
        {"$match": {
            "user_id": user_id, "type": "debit",
            "date": {"$gte": start, "$lte": end},
        }},
        {"$group": {"_id": None, "total": {"$sum": "$amount"}}},
    ]
    result = list(transactions_col().aggregate(pipeline))
    return Decimal(str(result[0]["total"])) if result else Decimal("0")
