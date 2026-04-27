"""Scheduler for morning/evening sync runs and the full sync pipeline."""

import asyncio
import logging
from datetime import datetime

from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger
import pytz

from config.settings import settings
from src.db.mongo import (
    insert_transaction, compute_dedup_hash, get_sync_state,
)
from src.sources.gmail_client import GmailClient
from src.parsers.email_parser import parse_emails_batch
from src.parsers.trade_parser import parse_trade_email
from src.parsers.subscription_parser import parse_subscription_email
from src.categorization.rules_engine import categorize_batch
from src.categorization.gemini_client import categorize_with_gemini

logger = logging.getLogger(__name__)

_scheduler: AsyncIOScheduler = None
_gmail_client: GmailClient = None


def get_scheduler() -> AsyncIOScheduler:
    global _scheduler
    if _scheduler is None:
        tz = pytz.timezone(settings.timezone)
        _scheduler = AsyncIOScheduler(timezone=tz)
    return _scheduler


def setup_schedule():
    """Configure morning and evening sync jobs."""
    scheduler = get_scheduler()
    tz = pytz.timezone(settings.timezone)

    morning_h, morning_m = settings.schedule_morning.split(":")
    evening_h, evening_m = settings.schedule_evening.split(":")

    scheduler.add_job(
        _morning_job,
        CronTrigger(hour=int(morning_h), minute=int(morning_m), timezone=tz),
        id="morning_sync",
        replace_existing=True,
    )

    scheduler.add_job(
        _evening_job,
        CronTrigger(hour=int(evening_h), minute=int(evening_m), timezone=tz),
        id="evening_sync",
        replace_existing=True,
    )

    logger.info("Scheduled morning sync at %s and evening sync at %s (%s)",
                settings.schedule_morning, settings.schedule_evening, settings.timezone)


async def _morning_job():
    logger.info("=== Morning sync started ===")
    try:
        count = await run_sync_pipeline()
        logger.info("Morning sync: processed %d transactions", count)

        from src.reports.telegram_bot import send_morning_report
        await send_morning_report()
    except Exception:
        logger.exception("Morning sync failed")
        try:
            from src.reports.telegram_bot import send_report
            await send_report("❌ Morning sync failed. Check logs for details.")
        except Exception:
            pass


async def _evening_job():
    logger.info("=== Evening sync started ===")
    try:
        count = await run_sync_pipeline()
        logger.info("Evening sync: processed %d transactions", count)

        from src.reports.telegram_bot import send_evening_report
        await send_evening_report()
    except Exception:
        logger.exception("Evening sync failed")
        try:
            from src.reports.telegram_bot import send_report
            await send_report("❌ Evening sync failed. Check logs for details.")
        except Exception:
            pass


async def run_sync_pipeline(user_id: str = "default") -> int:
    """Full sync pipeline: fetch emails → parse → categorize → store.

    Returns the number of new transactions stored.
    """
    global _gmail_client
    if _gmail_client is None:
        _gmail_client = GmailClient()

    total_stored = 0

    # 1. Fetch financial emails from Gmail
    logger.info("Step 1: Fetching emails from Gmail...")
    try:
        emails = _gmail_client.fetch_financial_emails(user_id=user_id)
        logger.info("Fetched %d emails", len(emails))
    except FileNotFoundError:
        logger.warning("Gmail not configured, skipping email sync")
        emails = []
    except Exception:
        logger.exception("Gmail fetch failed")
        emails = []

    if not emails:
        logger.info("No new emails to process")
        return 0

    # 2. Parse bank transaction emails
    logger.info("Step 2: Parsing transaction emails...")
    transactions = parse_emails_batch(emails)

    # 3. Also check for trade confirmations and subscriptions
    for email_data in emails:
        trade = parse_trade_email(email_data)
        if trade and trade.get("type") == "trade":
            _store_trade(trade, user_id)

        sub = parse_subscription_email(email_data)
        if sub:
            _store_subscription(sub, user_id)

    if not transactions:
        logger.info("No parseable transactions found")
        return 0

    # 4. Categorize
    logger.info("Step 3: Categorizing %d transactions...", len(transactions))
    categorized, uncategorized = categorize_batch(transactions)

    # 5. Send uncategorized to Gemini
    if uncategorized:
        logger.info("Step 4: Sending %d uncategorized to Gemini...", len(uncategorized))
        uncategorized = categorize_with_gemini(uncategorized)
        categorized.extend(uncategorized)

    # 6. Store
    logger.info("Step 5: Storing %d transactions...", len(categorized))
    for txn in categorized:
        if not txn.get("dedup_hash"):
            from decimal import Decimal
            txn["dedup_hash"] = compute_dedup_hash(
                txn["date"],
                txn.get("amount", Decimal("0")),
                txn.get("description", ""),
                txn.get("account_number_masked", ""),
            )
        txn["user_id"] = user_id

        result = insert_transaction(txn)
        if result:
            total_stored += 1

    # 7. Run trackers
    logger.info("Step 6: Running trackers...")
    try:
        from src.trackers.subscription_tracker import auto_register_subscriptions
        sub_count = auto_register_subscriptions(user_id)
        logger.info("Auto-detected %d subscriptions", sub_count)
    except Exception:
        logger.exception("Subscription detection failed")

    try:
        from src.trackers.investment_tracker import update_market_prices
        update_market_prices(user_id)
        logger.info("Market prices updated")
    except Exception:
        logger.exception("Market price update failed")

    try:
        from src.trackers.net_worth import calculate_net_worth
        calculate_net_worth(user_id)
        logger.info("Net worth snapshot saved")
    except Exception:
        logger.exception("Net worth calculation failed")

    logger.info("Sync complete: %d new transactions stored", total_stored)
    return total_stored


def _store_trade(trade: dict, user_id: str):
    """Store a trade confirmation in the investments collection."""
    from src.db.mongo import investments_col
    from decimal import Decimal

    if trade.get("trades"):
        for t in trade["trades"]:
            investments_col().update_one(
                {"user_id": user_id, "symbol": t.get("symbol", ""),
                 "broker": trade.get("broker", "")},
                {"$push": {"transactions": {
                    "date": trade["date"],
                    "type": t["trade_type"],
                    "units": str(t.get("units", 0)),
                    "price": str(t.get("price", 0)),
                }}},
                upsert=True,
            )
    elif trade.get("amount"):
        investments_col().update_one(
            {"user_id": user_id, "name": trade.get("security_name", ""),
             "broker": trade.get("broker", "")},
            {"$push": {"transactions": {
                "date": trade["date"],
                "type": trade.get("trade_type", "buy"),
                "units": str(trade.get("units", 0)),
                "price": str(trade.get("amount", 0)),
            }},
             "$set": {"type": "stock", "last_updated": datetime.utcnow()},
             "$setOnInsert": {
                 "user_id": user_id,
                 "name": trade.get("security_name", ""),
                 "broker": trade.get("broker", ""),
                 "created_at": datetime.utcnow(),
             }},
            upsert=True,
        )


def _store_subscription(sub: dict, user_id: str):
    """Store or update a subscription from email detection."""
    from src.db.mongo import subscriptions_col

    subscriptions_col().update_one(
        {"user_id": user_id, "name": sub["name"]},
        {"$set": {
            "amount": str(sub.get("amount", 0)) if sub.get("amount") else None,
            "frequency": sub.get("frequency", "monthly"),
            "category": sub.get("category", ""),
            "auto_detected": True,
            "status": "active",
        },
         "$setOnInsert": {
             "user_id": user_id,
             "name": sub["name"],
             "created_at": datetime.utcnow(),
         }},
        upsert=True,
    )
