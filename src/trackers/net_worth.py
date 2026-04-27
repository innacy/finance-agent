"""Aggregated net worth calculator -- combines all asset and liability trackers."""

import logging
from datetime import datetime
from decimal import Decimal

from src.db.mongo import (
    net_worth_col, _convert_decimals, _restore_decimals,
)
from src.trackers.account_tracker import get_total_bank_balance
from src.trackers.investment_tracker import get_portfolio_summary
from src.trackers.fd_rd_tracker import get_total_fd_rd_value
from src.trackers.loan_tracker import get_total_loan_outstanding
from src.trackers.credit_card_tracker import get_total_cc_dues

logger = logging.getLogger(__name__)


def calculate_net_worth(user_id: str = "default") -> dict:
    """Calculate current net worth from all tracked sources."""
    bank_balances = get_total_bank_balance(user_id)
    portfolio = get_portfolio_summary(user_id)
    investment_value = portfolio.get("total_current_value", Decimal("0"))
    fd_rd_value = get_total_fd_rd_value(user_id)
    loan_outstanding = get_total_loan_outstanding(user_id)
    cc_dues = get_total_cc_dues(user_id)

    total_assets = bank_balances + investment_value + fd_rd_value
    total_liabilities = loan_outstanding + cc_dues
    net_worth = total_assets - total_liabilities

    breakdown = {
        "bank_balances": bank_balances,
        "investments": investment_value,
        "fd_rd": fd_rd_value,
        "insurance_value": Decimal("0"),
        "loans_outstanding": loan_outstanding,
        "credit_card_dues": cc_dues,
    }

    snapshot = {
        "user_id": user_id,
        "date": datetime.utcnow(),
        "total_assets": total_assets,
        "total_liabilities": total_liabilities,
        "net_worth": net_worth,
        "breakdown": breakdown,
        "created_at": datetime.utcnow(),
    }

    net_worth_col().insert_one(_convert_decimals(snapshot))
    logger.info("Net worth snapshot: %s (assets: %s, liabilities: %s)",
                net_worth, total_assets, total_liabilities)

    return snapshot


def get_net_worth_trend(user_id: str = "default", days: int = 30) -> list[dict]:
    """Get daily net worth history."""
    from datetime import timedelta
    start = datetime.utcnow() - timedelta(days=days)
    cursor = net_worth_col().find(
        {"user_id": user_id, "date": {"$gte": start}}
    ).sort("date", 1)
    return [_restore_decimals(doc) for doc in cursor]


def get_net_worth_change(user_id: str = "default", days: int = 30) -> dict:
    """Compare current net worth with N days ago."""
    from datetime import timedelta
    now = datetime.utcnow()
    past_date = now - timedelta(days=days)

    current = net_worth_col().find_one(
        {"user_id": user_id}, sort=[("date", -1)]
    )
    past = net_worth_col().find_one(
        {"user_id": user_id, "date": {"$lte": past_date}},
        sort=[("date", -1)],
    )

    if not current:
        return {"current": Decimal("0"), "previous": Decimal("0"),
                "change": Decimal("0"), "percent": 0}

    current = _restore_decimals(current)
    current_nw = current.get("net_worth", Decimal("0"))

    if past:
        past = _restore_decimals(past)
        past_nw = past.get("net_worth", Decimal("0"))
    else:
        past_nw = Decimal("0")

    change = current_nw - past_nw
    pct = float(change / past_nw * 100) if past_nw != 0 else 0

    return {
        "current": current_nw,
        "previous": past_nw,
        "change": change,
        "percent": pct,
        "period_days": days,
    }
