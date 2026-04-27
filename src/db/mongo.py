"""MongoDB connection and collection management."""

import hashlib
import logging
from datetime import datetime
from decimal import Decimal
from typing import Any, Optional

from bson import Decimal128
from pymongo import MongoClient, IndexModel, ASCENDING, DESCENDING
from pymongo.collection import Collection
from pymongo.database import Database

from config.settings import settings

logger = logging.getLogger(__name__)

_client: Optional[MongoClient] = None
_db: Optional[Database] = None


def _convert_decimals(doc: dict) -> dict:
    """Convert Python Decimal values to BSON Decimal128 for MongoDB storage."""
    converted = {}
    for k, v in doc.items():
        if isinstance(v, Decimal):
            converted[k] = Decimal128(str(v))
        elif isinstance(v, dict):
            converted[k] = _convert_decimals(v)
        elif isinstance(v, list):
            converted[k] = [
                _convert_decimals(i) if isinstance(i, dict)
                else Decimal128(str(i)) if isinstance(i, Decimal)
                else i
                for i in v
            ]
        else:
            converted[k] = v
    return converted


def _restore_decimals(doc: dict) -> dict:
    """Convert BSON Decimal128 back to Python Decimal."""
    if doc is None:
        return doc
    restored = {}
    for k, v in doc.items():
        if isinstance(v, Decimal128):
            restored[k] = v.to_decimal()
        elif isinstance(v, dict):
            restored[k] = _restore_decimals(v)
        elif isinstance(v, list):
            restored[k] = [
                _restore_decimals(i) if isinstance(i, dict)
                else i.to_decimal() if isinstance(i, Decimal128)
                else i
                for i in v
            ]
        else:
            restored[k] = v
    return restored


def get_client() -> MongoClient:
    global _client
    if _client is None:
        _client = MongoClient(settings.mongo_uri)
        logger.info("Connected to MongoDB at %s", settings.mongo_uri)
    return _client


def get_db() -> Database:
    global _db
    if _db is None:
        _db = get_client()[settings.database_name]
    return _db


def close_connection():
    global _client, _db
    if _client:
        _client.close()
        _client = None
        _db = None
        logger.info("MongoDB connection closed")


# ── Collection accessors ─────────────────────────────────────────────────

def accounts_col() -> Collection:
    return get_db()["accounts"]


def transactions_col() -> Collection:
    return get_db()["transactions"]


def investments_col() -> Collection:
    return get_db()["investments"]


def fd_rd_col() -> Collection:
    return get_db()["fd_rd"]


def loans_credit_col() -> Collection:
    return get_db()["loans_credit"]


def insurance_col() -> Collection:
    return get_db()["insurance"]


def subscriptions_col() -> Collection:
    return get_db()["subscriptions"]


def net_worth_col() -> Collection:
    return get_db()["net_worth_snapshots"]


def sync_state_col() -> Collection:
    return get_db()["sync_state"]


# ── Index creation ───────────────────────────────────────────────────────

def ensure_indexes():
    """Create required indexes on all collections."""
    transactions_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("date", DESCENDING)]),
        IndexModel([("dedup_hash", ASCENDING)], unique=True, sparse=True),
        IndexModel([("user_id", ASCENDING), ("category", ASCENDING)]),
        IndexModel([("user_id", ASCENDING), ("merchant", ASCENDING)]),
        IndexModel([("user_id", ASCENDING), ("is_recurring", ASCENDING)]),
    ])

    accounts_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("bank_name", ASCENDING),
                     ("account_number_masked", ASCENDING)], unique=True),
    ])

    investments_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("type", ASCENDING),
                     ("symbol", ASCENDING)]),
    ])

    fd_rd_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("status", ASCENDING)]),
    ])

    loans_credit_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("type", ASCENDING)]),
    ])

    insurance_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("status", ASCENDING)]),
    ])

    subscriptions_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("status", ASCENDING)]),
    ])

    net_worth_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("date", DESCENDING)]),
    ])

    sync_state_col().create_indexes([
        IndexModel([("user_id", ASCENDING), ("source", ASCENDING)], unique=True),
    ])

    logger.info("MongoDB indexes ensured")


# ── CRUD helpers ─────────────────────────────────────────────────────────

def compute_dedup_hash(date: datetime, amount: Decimal, description: str,
                       account_masked: str) -> str:
    raw = f"{date.isoformat()}|{amount}|{description.strip().lower()}|{account_masked}"
    return hashlib.sha256(raw.encode()).hexdigest()[:32]


def insert_transaction(data: dict) -> Optional[str]:
    """Insert a transaction, skip if duplicate (dedup_hash collision)."""
    doc = _convert_decimals(data)
    try:
        result = transactions_col().insert_one(doc)
        return str(result.inserted_id)
    except Exception as e:
        if "duplicate key" in str(e).lower() or "E11000" in str(e):
            logger.debug("Duplicate transaction skipped: %s", doc.get("dedup_hash", ""))
            return None
        raise


def upsert_account(user_id: str, bank_name: str, account_masked: str,
                    update_fields: dict) -> None:
    doc = _convert_decimals(update_fields)
    accounts_col().update_one(
        {"user_id": user_id, "bank_name": bank_name,
         "account_number_masked": account_masked},
        {"$set": doc, "$setOnInsert": {
            "user_id": user_id, "bank_name": bank_name,
            "account_number_masked": account_masked,
            "created_at": datetime.utcnow(),
        }},
        upsert=True,
    )


def get_sync_state(user_id: str, source: str) -> Optional[dict]:
    doc = sync_state_col().find_one({"user_id": user_id, "source": source})
    return _restore_decimals(doc) if doc else None


def update_sync_state(user_id: str, source: str, **kwargs) -> None:
    sync_state_col().update_one(
        {"user_id": user_id, "source": source},
        {"$set": {**kwargs, "updated_at": datetime.utcnow()},
         "$setOnInsert": {"user_id": user_id, "source": source}},
        upsert=True,
    )


def find_transactions(user_id: str, filters: dict = None,
                      limit: int = 100, skip: int = 0) -> list[dict]:
    query = {"user_id": user_id}
    if filters:
        query.update(filters)
    cursor = (transactions_col()
              .find(query)
              .sort("date", DESCENDING)
              .skip(skip)
              .limit(limit))
    return [_restore_decimals(doc) for doc in cursor]


def aggregate_spending(user_id: str, start_date: datetime,
                       end_date: datetime) -> list[dict]:
    pipeline = [
        {"$match": {
            "user_id": user_id,
            "type": "debit",
            "date": {"$gte": start_date, "$lte": end_date},
        }},
        {"$group": {
            "_id": "$category",
            "total": {"$sum": "$amount"},
            "count": {"$sum": 1},
        }},
        {"$sort": {"total": -1}},
    ]
    return list(transactions_col().aggregate(pipeline))
