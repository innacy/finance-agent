"""Track loans (home, car, personal, education) -- EMI, principal, interest."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from src.db.mongo import loans_credit_col, _restore_decimals, _convert_decimals

logger = logging.getLogger(__name__)


def add_loan(user_id: str, loan_type: str, lender: str,
             total_amount: Decimal, interest_rate: float,
             emi_amount: Decimal, emi_date: int, tenure_months: int,
             account_masked: str = "") -> str:
    """Add a new loan entry."""
    remaining = tenure_months
    outstanding = total_amount

    doc = _convert_decimals({
        "user_id": user_id,
        "type": loan_type,
        "lender": lender,
        "account_masked": account_masked,
        "total_amount": total_amount,
        "outstanding": outstanding,
        "emi_amount": emi_amount,
        "interest_rate": interest_rate,
        "emi_date": emi_date,
        "tenure_months": tenure_months,
        "remaining_emis": remaining,
        "next_due_date": _next_emi_date(emi_date),
        "payment_history": [],
        "created_at": datetime.utcnow(),
        "updated_at": datetime.utcnow(),
    })
    result = loans_credit_col().insert_one(doc)
    return str(result.inserted_id)


def record_emi_payment(user_id: str, lender: str, amount: Decimal,
                        principal_component: Optional[Decimal] = None,
                        interest_component: Optional[Decimal] = None):
    """Record an EMI payment and update outstanding balance."""
    loan = loans_credit_col().find_one(
        {"user_id": user_id, "lender": lender, "type": {"$ne": "credit_card"}}
    )
    if not loan:
        logger.warning("Loan not found for lender: %s", lender)
        return

    loan = _restore_decimals(loan)
    outstanding = loan.get("outstanding", Decimal("0"))

    if principal_component is None:
        rate = loan.get("interest_rate", 0) / 100 / 12
        interest_component = outstanding * Decimal(str(rate))
        principal_component = amount - interest_component

    new_outstanding = max(outstanding - principal_component, Decimal("0"))
    remaining = max(loan.get("remaining_emis", 0) - 1, 0)

    payment = _convert_decimals({
        "date": datetime.utcnow(),
        "amount": amount,
        "principal": principal_component,
        "interest": interest_component,
    })

    loans_credit_col().update_one(
        {"_id": loan["_id"]},
        {
            "$set": _convert_decimals({
                "outstanding": new_outstanding,
                "remaining_emis": remaining,
                "next_due_date": _next_emi_date(loan.get("emi_date", 1)),
                "updated_at": datetime.utcnow(),
            }),
            "$push": {"payment_history": payment},
        },
    )


def get_total_loan_outstanding(user_id: str = "default") -> Decimal:
    pipeline = [
        {"$match": {"user_id": user_id, "type": {"$ne": "credit_card"}}},
        {"$group": {"_id": None, "total": {"$sum": "$outstanding"}}},
    ]
    result = list(loans_credit_col().aggregate(pipeline))
    if result:
        return Decimal(str(result[0].get("total", 0)))
    return Decimal("0")


def get_upcoming_emis(user_id: str, days: int = 30) -> list[dict]:
    cutoff = datetime.utcnow() + timedelta(days=days)
    loans = loans_credit_col().find({
        "user_id": user_id,
        "next_due_date": {"$lte": cutoff},
        "remaining_emis": {"$gt": 0},
    })
    return [_restore_decimals(l) for l in loans]


def _next_emi_date(emi_day: int) -> datetime:
    now = datetime.utcnow()
    if now.day < emi_day:
        return now.replace(day=min(emi_day, 28))
    else:
        month = now.month + 1
        year = now.year
        if month > 12:
            month = 1
            year += 1
        return datetime(year, month, min(emi_day, 28))
