"""Track bank account balances from transaction data."""

import logging
from datetime import datetime
from decimal import Decimal

from src.db.mongo import (
    accounts_col, transactions_col, upsert_account,
    _restore_decimals, _convert_decimals,
)

logger = logging.getLogger(__name__)


def update_account_from_transaction(txn: dict, user_id: str = "default"):
    """Update account balance based on a single transaction."""
    bank = txn.get("bank_name", "")
    acc_masked = txn.get("account_number_masked", "")
    if not bank or not acc_masked:
        return

    amount = txn.get("amount", Decimal("0"))
    if isinstance(amount, (int, float)):
        amount = Decimal(str(amount))

    if txn.get("type") == "debit":
        update = {"$inc": {"current_balance": -float(amount)}}
    else:
        update = {"$inc": {"current_balance": float(amount)}}

    accounts_col().update_one(
        {"user_id": user_id, "bank_name": bank,
         "account_number_masked": acc_masked},
        {**update, "$set": {"last_synced": datetime.utcnow(), "updated_at": datetime.utcnow()},
         "$setOnInsert": {
             "user_id": user_id, "bank_name": bank,
             "account_number_masked": acc_masked,
             "account_type": "savings",
             "created_at": datetime.utcnow(),
         }},
        upsert=True,
    )


def set_account_balance(user_id: str, bank_name: str, account_masked: str,
                         balance: Decimal):
    """Directly set account balance (from statement ending balance)."""
    upsert_account(user_id, bank_name, account_masked, {
        "current_balance": balance,
        "last_synced": datetime.utcnow(),
        "updated_at": datetime.utcnow(),
    })


def get_total_bank_balance(user_id: str = "default") -> Decimal:
    """Sum of all account balances."""
    pipeline = [
        {"$match": {"user_id": user_id}},
        {"$group": {"_id": None, "total": {"$sum": "$current_balance"}}},
    ]
    result = list(accounts_col().aggregate(pipeline))
    if result:
        total = result[0].get("total", 0)
        return Decimal(str(total))
    return Decimal("0")


def get_all_accounts(user_id: str = "default") -> list[dict]:
    return [_restore_decimals(a) for a in accounts_col().find({"user_id": user_id})]
