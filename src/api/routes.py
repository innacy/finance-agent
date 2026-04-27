"""FastAPI routes for iData-ui consumption and statement upload."""

import logging
import os
from datetime import datetime, timedelta
from decimal import Decimal
from typing import Optional

from fastapi import APIRouter, File, UploadFile, Query, HTTPException

from config.settings import settings
from src.db.mongo import (
    accounts_col, find_transactions, aggregate_spending,
    investments_col, subscriptions_col, net_worth_col,
    loans_credit_col, insurance_col, fd_rd_col,
    insert_transaction, compute_dedup_hash,
)
from src.db.mongo import _restore_decimals
from src.sources.statement_parser import StatementParser
from src.parsers.statement_extractor import normalize_statement_transactions
from src.categorization.rules_engine import categorize_transaction
from src.categorization.gemini_client import categorize_with_gemini
from src.api.schemas import (
    SyncResponse, UploadResponse, SpendingSummary, SpendingCategory,
)
from src.categorization.categories import CATEGORY_DISPLAY_NAMES

logger = logging.getLogger(__name__)
router = APIRouter()


@router.get("/accounts")
async def get_accounts():
    accounts = [_restore_decimals(a) for a in accounts_col().find({"user_id": "default"})]
    for a in accounts:
        a.pop("_id", None)
        bal = a.get("current_balance", Decimal("0"))
        a["current_balance"] = float(bal) if isinstance(bal, Decimal) else bal
    return accounts


@router.get("/transactions")
async def get_transactions(
    days: int = Query(30, ge=1, le=365),
    category: Optional[str] = None,
    limit: int = Query(100, ge=1, le=1000),
    skip: int = Query(0, ge=0),
):
    filters = {}
    if category:
        filters["category"] = category
    start = datetime.utcnow() - timedelta(days=days)
    filters["date"] = {"$gte": start}

    txns = find_transactions("default", filters, limit=limit, skip=skip)
    for t in txns:
        t.pop("_id", None)
        t.pop("raw_data", None)
        if isinstance(t.get("amount"), Decimal):
            t["amount"] = float(t["amount"])
    return txns


@router.get("/spending")
async def get_spending(
    days: int = Query(30, ge=1, le=365),
) -> SpendingSummary:
    end = datetime.utcnow()
    start = end - timedelta(days=days)
    raw = aggregate_spending("default", start, end)

    categories = []
    total = 0.0
    for item in raw:
        amt = float(item.get("total", 0))
        total += amt
        categories.append(SpendingCategory(
            category=item["_id"],
            display_name=CATEGORY_DISPLAY_NAMES.get(item["_id"], item["_id"]),
            total=amt,
            count=item.get("count", 0),
        ))

    return SpendingSummary(
        period_start=start, period_end=end,
        total_spending=total, categories=categories,
    )


@router.get("/networth")
async def get_networth():
    latest = net_worth_col().find_one({"user_id": "default"}, sort=[("date", -1)])
    if not latest:
        return {"message": "No net worth data yet. Run a sync first."}
    doc = _restore_decimals(latest)
    doc.pop("_id", None)
    for key in ["total_assets", "total_liabilities", "net_worth"]:
        if isinstance(doc.get(key), Decimal):
            doc[key] = float(doc[key])
    bd = doc.get("breakdown", {})
    for k, v in bd.items():
        if isinstance(v, Decimal):
            bd[k] = float(v)
    return doc


@router.get("/networth/history")
async def get_networth_history(days: int = Query(30, ge=1, le=365)):
    start = datetime.utcnow() - timedelta(days=days)
    cursor = net_worth_col().find(
        {"user_id": "default", "date": {"$gte": start}},
    ).sort("date", 1)
    results = []
    for doc in cursor:
        doc = _restore_decimals(doc)
        doc.pop("_id", None)
        for key in ["total_assets", "total_liabilities", "net_worth"]:
            if isinstance(doc.get(key), Decimal):
                doc[key] = float(doc[key])
        results.append(doc)
    return results


@router.get("/investments")
async def get_investments():
    investments = list(investments_col().find({"user_id": "default"}))
    results = []
    for inv in investments:
        inv = _restore_decimals(inv)
        inv.pop("_id", None)
        inv.pop("transactions", None)
        for key in ["units", "avg_buy_price", "current_price", "current_value",
                     "invested_amount", "pnl"]:
            if isinstance(inv.get(key), Decimal):
                inv[key] = float(inv[key])
        results.append(inv)
    return results


@router.get("/subscriptions")
async def get_subscriptions():
    subs = list(subscriptions_col().find({"user_id": "default"}))
    results = []
    for sub in subs:
        sub = _restore_decimals(sub)
        sub.pop("_id", None)
        if isinstance(sub.get("amount"), Decimal):
            sub["amount"] = float(sub["amount"])
        results.append(sub)
    return results


@router.get("/loans")
async def get_loans():
    loans = list(loans_credit_col().find({"user_id": "default"}))
    results = []
    for loan in loans:
        loan = _restore_decimals(loan)
        loan.pop("_id", None)
        loan.pop("payment_history", None)
        for key in ["total_amount", "outstanding", "emi_amount"]:
            if isinstance(loan.get(key), Decimal):
                loan[key] = float(loan[key])
        results.append(loan)
    return results


@router.get("/insurance")
async def get_insurance():
    policies = list(insurance_col().find({"user_id": "default"}))
    results = []
    for pol in policies:
        pol = _restore_decimals(pol)
        pol.pop("_id", None)
        for key in ["premium_amount", "sum_assured"]:
            if isinstance(pol.get(key), Decimal):
                pol[key] = float(pol[key])
        results.append(pol)
    return results


@router.get("/fd-rd")
async def get_fd_rd():
    items = list(fd_rd_col().find({"user_id": "default"}))
    results = []
    for item in items:
        item = _restore_decimals(item)
        item.pop("_id", None)
        for key in ["principal", "maturity_amount", "monthly_installment"]:
            if isinstance(item.get(key), Decimal):
                item[key] = float(item[key])
        results.append(item)
    return results


@router.post("/sync")
async def trigger_sync() -> SyncResponse:
    try:
        from src.scheduler import run_sync_pipeline
        count = await run_sync_pipeline()
        return SyncResponse(
            status="success",
            transactions_processed=count,
            message=f"Sync complete. {count} new transactions processed.",
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/spending/trend")
async def get_spending_trend(months: int = Query(6, ge=1, le=24)):
    from src.trackers.analytics import monthly_spending_trend
    trend = monthly_spending_trend("default", months)
    for item in trend:
        item["total_spending"] = float(item["total_spending"])
        item.pop("month_start", None)
    return trend


@router.get("/spending/insights")
async def get_spending_insights():
    from src.trackers.analytics import spending_insights, month_over_month_change
    insights = spending_insights("default")
    mom = month_over_month_change("default")
    for key in ["this_month", "last_month", "change"]:
        if isinstance(mom.get(key), Decimal):
            mom[key] = float(mom[key])
    return {"insights": insights, "month_over_month": mom}


@router.get("/spending/merchants")
async def get_top_merchants(days: int = Query(30, ge=1, le=365),
                             limit: int = Query(10, ge=1, le=50)):
    from src.trackers.analytics import top_merchants
    merchants = top_merchants("default", days, limit)
    for m in merchants:
        m["merchant"] = m.pop("_id", "")
        m["total"] = float(m["total"])
    return merchants


@router.get("/spending/unusual")
async def get_unusual_transactions():
    from src.trackers.analytics import detect_unusual_transactions
    unusual = detect_unusual_transactions("default")
    for u in unusual:
        if isinstance(u.get("amount"), Decimal):
            u["amount"] = float(u["amount"])
        if isinstance(u.get("avg_for_category"), Decimal):
            u["avg_for_category"] = float(u["avg_for_category"])
    return unusual


@router.post("/statements/upload")
async def upload_statement(
    file: UploadFile = File(...),
    bank_name: str = Query("hdfc"),
    account_masked: str = Query(""),
) -> UploadResponse:
    """Upload a bank statement (PDF/CSV) for historical import."""
    if not file.filename:
        raise HTTPException(status_code=400, detail="No file provided")

    ext = file.filename.rsplit(".", 1)[-1].lower()
    if ext not in ("pdf", "csv"):
        raise HTTPException(status_code=400, detail="Only PDF and CSV files accepted")

    content = await file.read()
    if len(content) > settings.max_upload_size_mb * 1024 * 1024:
        raise HTTPException(status_code=400, detail="File too large")

    # Save file
    os.makedirs(settings.upload_dir, exist_ok=True)
    save_path = os.path.join(settings.upload_dir, file.filename)
    with open(save_path, "wb") as f:
        f.write(content)

    # Parse
    parser = StatementParser()
    raw_txns = parser.parse_file(save_path, bank_name)
    normalized = normalize_statement_transactions(raw_txns, bank_name.upper(), account_masked)

    stored = 0
    duplicates = 0
    for txn in normalized:
        txn = categorize_transaction(txn)
        txn["user_id"] = "default"
        if not txn.get("dedup_hash"):
            txn["dedup_hash"] = compute_dedup_hash(
                txn["date"], txn["amount"], txn["description"],
                txn.get("account_number_masked", ""),
            )
        result = insert_transaction(txn)
        if result:
            stored += 1
        else:
            duplicates += 1

    # Categorize uncategorized with Gemini
    uncategorized_txns = [t for t in normalized if t.get("categorized_by") == "uncategorized"]
    if uncategorized_txns:
        categorize_with_gemini(uncategorized_txns)

    return UploadResponse(
        status="success",
        transactions_found=len(normalized),
        transactions_stored=stored,
        duplicates_skipped=duplicates,
    )


@router.post("/statements/upload-cas")
async def upload_cas(file: UploadFile = File(...)):
    """Upload a CAMS/KFintech CAS PDF for mutual fund portfolio import."""
    if not file.filename or not file.filename.lower().endswith(".pdf"):
        raise HTTPException(status_code=400, detail="Only PDF files accepted for CAS")

    content = await file.read()
    os.makedirs(settings.upload_dir, exist_ok=True)
    save_path = os.path.join(settings.upload_dir, f"cas_{file.filename}")
    with open(save_path, "wb") as f:
        f.write(content)

    from src.parsers.cas_parser import CASParser, import_cas_to_investments
    parser = CASParser()
    data = parser.parse(save_path)
    count = import_cas_to_investments(data)

    return {
        "status": "success",
        "investor": data.get("investor_name", ""),
        "schemes_imported": count,
        "total_value": float(data.get("total_value", 0)),
    }
