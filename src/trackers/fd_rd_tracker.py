"""Track Fixed Deposits and Recurring Deposits."""

import logging
from datetime import datetime
from decimal import Decimal
from typing import Optional

from src.db.mongo import fd_rd_col, _restore_decimals, _convert_decimals

logger = logging.getLogger(__name__)


def add_fd(user_id: str, bank_name: str, principal: Decimal,
           interest_rate: float, maturity_date: datetime,
           maturity_amount: Optional[Decimal] = None,
           start_date: Optional[datetime] = None) -> str:
    """Register a Fixed Deposit."""
    if maturity_amount is None:
        years = (maturity_date - (start_date or datetime.utcnow())).days / 365
        maturity_amount = principal * (1 + Decimal(str(interest_rate / 100))) ** Decimal(str(years))

    doc = _convert_decimals({
        "user_id": user_id,
        "type": "fd",
        "bank_name": bank_name,
        "principal": principal,
        "interest_rate": interest_rate,
        "start_date": start_date or datetime.utcnow(),
        "maturity_date": maturity_date,
        "maturity_amount": maturity_amount,
        "monthly_installment": Decimal("0"),
        "status": "active",
        "created_at": datetime.utcnow(),
    })
    result = fd_rd_col().insert_one(doc)
    return str(result.inserted_id)


def add_rd(user_id: str, bank_name: str, monthly_installment: Decimal,
           interest_rate: float, maturity_date: datetime,
           start_date: Optional[datetime] = None,
           maturity_amount: Optional[Decimal] = None) -> str:
    """Register a Recurring Deposit."""
    months = max(1, (maturity_date - (start_date or datetime.utcnow())).days // 30)
    principal = monthly_installment * months

    doc = _convert_decimals({
        "user_id": user_id,
        "type": "rd",
        "bank_name": bank_name,
        "principal": principal,
        "interest_rate": interest_rate,
        "start_date": start_date or datetime.utcnow(),
        "maturity_date": maturity_date,
        "maturity_amount": maturity_amount or principal,
        "monthly_installment": monthly_installment,
        "status": "active",
        "created_at": datetime.utcnow(),
    })
    result = fd_rd_col().insert_one(doc)
    return str(result.inserted_id)


def get_total_fd_rd_value(user_id: str = "default") -> Decimal:
    """Current value of all active FDs and RDs (estimated as principal + accrued interest)."""
    total = Decimal("0")
    now = datetime.utcnow()

    for item in fd_rd_col().find({"user_id": user_id, "status": "active"}):
        item = _restore_decimals(item)
        principal = item.get("principal", Decimal("0"))
        rate = item.get("interest_rate", 0) / 100
        start = item.get("start_date") or item.get("created_at", now)
        elapsed_years = max(0, (now - start).days) / 365

        if item.get("type") == "fd":
            current_value = principal * (1 + Decimal(str(rate))) ** Decimal(str(elapsed_years))
        else:
            monthly = item.get("monthly_installment", Decimal("0"))
            months_elapsed = max(1, int(elapsed_years * 12))
            deposited = monthly * months_elapsed
            monthly_rate = Decimal(str(rate / 12))
            interest = deposited * monthly_rate * Decimal(str(months_elapsed / 2))
            current_value = deposited + interest

        total += current_value

    return total


def check_maturing_soon(user_id: str, days: int = 30) -> list[dict]:
    from datetime import timedelta
    cutoff = datetime.utcnow() + timedelta(days=days)
    items = fd_rd_col().find({
        "user_id": user_id,
        "status": "active",
        "maturity_date": {"$lte": cutoff, "$gte": datetime.utcnow()},
    })
    return [_restore_decimals(i) for i in items]


def mark_matured(user_id: str, bank_name: str, fd_type: str = "fd"):
    fd_rd_col().update_many(
        {"user_id": user_id, "bank_name": bank_name, "type": fd_type,
         "maturity_date": {"$lte": datetime.utcnow()}, "status": "active"},
        {"$set": {"status": "matured"}},
    )
