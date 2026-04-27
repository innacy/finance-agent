"""Track credit card bills, dues, and spending."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from src.db.mongo import loans_credit_col, transactions_col, _restore_decimals, _convert_decimals

logger = logging.getLogger(__name__)


def add_credit_card(user_id: str, lender: str, account_masked: str,
                     credit_limit: Decimal, billing_date: int = 1) -> str:
    """Register a credit card for tracking."""
    doc = _convert_decimals({
        "user_id": user_id,
        "type": "credit_card",
        "lender": lender,
        "account_masked": account_masked,
        "total_amount": credit_limit,
        "outstanding": Decimal("0"),
        "emi_amount": Decimal("0"),
        "interest_rate": 0.0,
        "emi_date": billing_date,
        "tenure_months": 0,
        "remaining_emis": 0,
        "next_due_date": None,
        "payment_history": [],
        "created_at": datetime.utcnow(),
        "updated_at": datetime.utcnow(),
    })
    result = loans_credit_col().insert_one(doc)
    return str(result.inserted_id)


def update_cc_outstanding(user_id: str, lender: str, outstanding: Decimal,
                           due_date: Optional[datetime] = None):
    """Update credit card outstanding amount (from statement or alert)."""
    update = _convert_decimals({
        "outstanding": outstanding,
        "updated_at": datetime.utcnow(),
    })
    if due_date:
        update["next_due_date"] = due_date

    loans_credit_col().update_one(
        {"user_id": user_id, "type": "credit_card", "lender": lender},
        {"$set": update},
    )


def record_cc_payment(user_id: str, lender: str, amount: Decimal):
    """Record a credit card bill payment."""
    card = loans_credit_col().find_one(
        {"user_id": user_id, "type": "credit_card", "lender": lender}
    )
    if not card:
        return

    card = _restore_decimals(card)
    new_outstanding = max(card.get("outstanding", Decimal("0")) - amount, Decimal("0"))

    payment = _convert_decimals({
        "date": datetime.utcnow(),
        "amount": amount,
        "principal": amount,
        "interest": Decimal("0"),
    })

    loans_credit_col().update_one(
        {"_id": card["_id"]},
        {
            "$set": _convert_decimals({
                "outstanding": new_outstanding,
                "updated_at": datetime.utcnow(),
            }),
            "$push": {"payment_history": payment},
        },
    )


def get_total_cc_dues(user_id: str = "default") -> Decimal:
    pipeline = [
        {"$match": {"user_id": user_id, "type": "credit_card"}},
        {"$group": {"_id": None, "total": {"$sum": "$outstanding"}}},
    ]
    result = list(loans_credit_col().aggregate(pipeline))
    if result:
        return Decimal(str(result[0].get("total", 0)))
    return Decimal("0")


def get_cc_spending_by_category(user_id: str, days: int = 30) -> list[dict]:
    """Get credit card spending breakdown by category."""
    start = datetime.utcnow() - timedelta(days=days)
    pipeline = [
        {"$match": {
            "user_id": user_id,
            "payment_mode": "card",
            "type": "debit",
            "date": {"$gte": start},
        }},
        {"$group": {
            "_id": "$category",
            "total": {"$sum": "$amount"},
            "count": {"$sum": 1},
        }},
        {"$sort": {"total": -1}},
    ]
    return list(transactions_col().aggregate(pipeline))
