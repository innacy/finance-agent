"""Generate morning and evening financial reports."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal

from src.db.mongo import (
    accounts_col, transactions_col, aggregate_spending,
    loans_credit_col, insurance_col, subscriptions_col,
    net_worth_col, investments_col, fd_rd_col,
)
from src.db.mongo import _restore_decimals
from src.reports.formatters import (
    format_account_summary, format_spending_summary,
    format_upcoming_dues, format_net_worth, format_currency,
)

logger = logging.getLogger(__name__)


def generate_morning_report(user_id: str = "default") -> str:
    """Morning report: yesterday's spending, balances, upcoming dues."""
    now = datetime.utcnow()
    yesterday_start = (now - timedelta(days=1)).replace(hour=0, minute=0, second=0)
    yesterday_end = yesterday_start.replace(hour=23, minute=59, second=59)

    parts = ["☀️ *Good Morning! Daily Finance Report*\n"]
    parts.append(f"📆 {now.strftime('%A, %d %B %Y')}\n")

    # Account balances
    accounts = [_restore_decimals(a) for a in accounts_col().find({"user_id": user_id})]
    parts.append(format_account_summary(accounts))
    parts.append("")

    # Yesterday's spending
    spending = aggregate_spending(user_id, yesterday_start, yesterday_end)
    if spending:
        parts.append("📊 *Yesterday's Spending*\n")
        parts.append(format_spending_summary(spending))
    else:
        parts.append("📊 No spending recorded yesterday.")
    parts.append("")

    # Upcoming dues (next 7 days)
    upcoming = _get_upcoming_dues(user_id, days=7)
    parts.append(format_upcoming_dues(upcoming))
    parts.append("")

    # Subscription renewals this week
    subs = _get_upcoming_subscriptions(user_id, days=7)
    if subs:
        parts.append("🔄 *Subscription Renewals This Week*\n")
        for sub in subs:
            parts.append(
                f"  • {sub['name']}: {format_currency(sub.get('amount', Decimal('0')))} "
                f"({sub.get('frequency', 'monthly')})"
            )

    return "\n".join(parts)


def generate_evening_report(user_id: str = "default") -> str:
    """Evening report: today's spending, net worth, portfolio P&L."""
    now = datetime.utcnow()
    today_start = now.replace(hour=0, minute=0, second=0)

    parts = ["🌙 *Evening Finance Update*\n"]
    parts.append(f"📆 {now.strftime('%A, %d %B %Y')}\n")

    # Today's spending
    spending = aggregate_spending(user_id, today_start, now)
    if spending:
        parts.append("💸 *Today's Spending*\n")
        parts.append(format_spending_summary(spending))
    else:
        parts.append("💸 No spending recorded today.")
    parts.append("")

    # Net worth
    latest_nw = net_worth_col().find_one(
        {"user_id": user_id}, sort=[("date", -1)]
    )
    if latest_nw:
        parts.append(format_net_worth(_restore_decimals(latest_nw)))
    parts.append("")

    # Investment P&L
    investments = list(investments_col().find({"user_id": user_id}))
    if investments:
        parts.append("📈 *Investment Portfolio*\n")
        total_invested = Decimal("0")
        total_current = Decimal("0")
        for inv in investments:
            inv = _restore_decimals(inv)
            invested = inv.get("invested_amount", Decimal("0"))
            current = inv.get("current_value", Decimal("0"))
            total_invested += invested
            total_current += current

        pnl = total_current - total_invested
        pnl_pct = float(pnl / total_invested * 100) if total_invested else 0
        arrow = "📈" if pnl > 0 else "📉"
        parts.append(f"  Invested: {format_currency(total_invested)}")
        parts.append(f"  Current: {format_currency(total_current)}")
        parts.append(f"  {arrow} P&L: {format_currency(pnl)} ({pnl_pct:+.2f}%)")
    parts.append("")

    # Unusual spending alert
    alert = _check_unusual_spending(user_id, today_start, now)
    if alert:
        parts.append(f"⚠️ *Alert*: {alert}")

    return "\n".join(parts)


def _get_upcoming_dues(user_id: str, days: int = 7) -> list[dict]:
    now = datetime.utcnow()
    cutoff = now + timedelta(days=days)
    dues = []

    # Loan EMIs
    for loan in loans_credit_col().find({"user_id": user_id}):
        loan = _restore_decimals(loan)
        due = loan.get("next_due_date")
        if due and now <= due <= cutoff:
            dues.append({
                "name": loan.get("lender", "Loan"),
                "type": loan.get("type", "loan"),
                "amount": loan.get("emi_amount", Decimal("0")),
                "due_date": due,
            })

    # Insurance premiums
    for ins in insurance_col().find({"user_id": user_id, "status": "active"}):
        ins = _restore_decimals(ins)
        due = ins.get("next_premium_date")
        if due and now <= due <= cutoff:
            dues.append({
                "name": ins.get("provider", "Insurance"),
                "type": f"{ins.get('type', '')} insurance",
                "amount": ins.get("premium_amount", Decimal("0")),
                "due_date": due,
            })

    dues.sort(key=lambda x: x.get("due_date", now))
    return dues


def _get_upcoming_subscriptions(user_id: str, days: int = 7) -> list[dict]:
    now = datetime.utcnow()
    cutoff = now + timedelta(days=days)
    subs = []
    for sub in subscriptions_col().find({"user_id": user_id, "status": "active"}):
        sub = _restore_decimals(sub)
        billing = sub.get("next_billing_date")
        if billing and now <= billing <= cutoff:
            subs.append(sub)
    return subs


def _check_unusual_spending(user_id: str, start: datetime, end: datetime) -> str:
    """Check if today's spending is significantly above the 30-day average."""
    today_total = Decimal("0")
    for cat in aggregate_spending(user_id, start, end):
        today_total += cat.get("total", Decimal("0"))

    avg_start = start - timedelta(days=30)
    month_spending = aggregate_spending(user_id, avg_start, start)
    month_total = sum(c.get("total", Decimal("0")) for c in month_spending)
    daily_avg = month_total / 30 if month_total else Decimal("0")

    if daily_avg > 0 and today_total > daily_avg * 2:
        return (
            f"Today's spending ({format_currency(today_total)}) is "
            f"{float(today_total / daily_avg):.1f}x your daily average "
            f"({format_currency(daily_avg)})"
        )
    return ""
