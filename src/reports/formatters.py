"""Rich text formatting helpers for Telegram messages."""

from decimal import Decimal
from typing import Optional

from src.categorization.categories import CATEGORY_DISPLAY_NAMES


def format_currency(amount: Optional[Decimal], symbol: str = "₹") -> str:
    if amount is None:
        return f"{symbol}0"
    sign = "-" if amount < 0 else ""
    abs_amt = abs(amount)
    if abs_amt >= 10_000_000:
        return f"{sign}{symbol}{abs_amt / 10_000_000:.2f} Cr"
    elif abs_amt >= 100_000:
        return f"{sign}{symbol}{abs_amt / 100_000:.2f} L"
    elif abs_amt >= 1000:
        return f"{sign}{symbol}{abs_amt:,.0f}"
    else:
        return f"{sign}{symbol}{abs_amt:,.2f}"


def format_percent(value: float) -> str:
    sign = "+" if value > 0 else ""
    return f"{sign}{value:.2f}%"


def format_change(current: Decimal, previous: Decimal) -> str:
    if previous == 0:
        return ""
    change = current - previous
    pct = float(change / previous * 100)
    arrow = "📈" if change > 0 else "📉" if change < 0 else "➡️"
    return f"{arrow} {format_currency(change)} ({format_percent(pct)})"


def category_display(cat: str) -> str:
    return CATEGORY_DISPLAY_NAMES.get(cat, cat.replace("_", " ").title())


def spending_bar(amount: Decimal, max_amount: Decimal, width: int = 10) -> str:
    if max_amount == 0:
        return "░" * width
    ratio = min(float(amount / max_amount), 1.0)
    filled = int(ratio * width)
    return "█" * filled + "░" * (width - filled)


def format_account_summary(accounts: list[dict]) -> str:
    if not accounts:
        return "No accounts tracked yet."
    lines = ["🏦 *Account Balances*\n"]
    total = Decimal("0")
    for acc in accounts:
        bal = acc.get("current_balance", Decimal("0"))
        if isinstance(bal, (int, float)):
            bal = Decimal(str(bal))
        total += bal
        lines.append(
            f"  {acc['bank_name']} ({acc.get('account_number_masked', '')}) "
            f"→ {format_currency(bal)}"
        )
    lines.append(f"\n  *Total*: {format_currency(total)}")
    return "\n".join(lines)


def format_spending_summary(categories: list[dict]) -> str:
    if not categories:
        return "No spending recorded."
    lines = ["💸 *Spending Breakdown*\n"]
    max_amt = max(
        (c.get("total", Decimal("0")) for c in categories),
        default=Decimal("0"),
    )
    if isinstance(max_amt, (int, float)):
        max_amt = Decimal(str(max_amt))
    total = Decimal("0")
    for cat in categories[:10]:
        amt = cat.get("total", Decimal("0"))
        if isinstance(amt, (int, float)):
            amt = Decimal(str(amt))
        total += amt
        bar = spending_bar(amt, max_amt)
        name = category_display(cat.get("_id", "others"))
        count = cat.get("count", 0)
        lines.append(f"  {bar} {name}: {format_currency(amt)} ({count})")
    lines.append(f"\n  *Total*: {format_currency(total)}")
    return "\n".join(lines)


def format_upcoming_dues(items: list[dict]) -> str:
    if not items:
        return "✅ No upcoming dues this week."
    lines = ["📅 *Upcoming Dues*\n"]
    for item in items:
        due_date = item.get("due_date", "")
        if hasattr(due_date, "strftime"):
            due_date = due_date.strftime("%d %b")
        name = item.get("name", "Unknown")
        amount = item.get("amount", Decimal("0"))
        item_type = item.get("type", "")
        lines.append(f"  • {due_date} — {name} ({item_type}): {format_currency(amount)}")
    return "\n".join(lines)


def format_net_worth(snapshot: dict) -> str:
    if not snapshot:
        return "Net worth not calculated yet."
    nw = snapshot.get("net_worth", Decimal("0"))
    bd = snapshot.get("breakdown", {})
    lines = [
        f"💰 *Net Worth*: {format_currency(nw)}\n",
        f"  🏦 Banks: {format_currency(bd.get('bank_balances', Decimal('0')))}",
        f"  📊 Investments: {format_currency(bd.get('investments', Decimal('0')))}",
        f"  🏛️ FD/RD: {format_currency(bd.get('fd_rd', Decimal('0')))}",
        f"  🔴 Loans: -{format_currency(bd.get('loans_outstanding', Decimal('0')))}",
        f"  💳 CC Dues: -{format_currency(bd.get('credit_card_dues', Decimal('0')))}",
    ]
    return "\n".join(lines)
