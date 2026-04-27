"""Telegram bot for sending reports and handling on-demand commands."""

import logging
from datetime import datetime, timedelta
from decimal import Decimal

from telegram import Update
from telegram.ext import (
    Application, CommandHandler, ContextTypes,
)

from config.settings import settings
from src.db.mongo import (
    accounts_col, aggregate_spending, investments_col,
    subscriptions_col, net_worth_col, loans_credit_col,
    insurance_col,
)
from src.db.mongo import _restore_decimals
from src.reports.formatters import (
    format_account_summary, format_spending_summary,
    format_net_worth, format_upcoming_dues, format_currency,
)
from src.reports.daily_report import (
    generate_morning_report, generate_evening_report,
    _get_upcoming_dues,
)

logger = logging.getLogger(__name__)

_app: Application = None


def get_bot_app() -> Application:
    global _app
    if _app is None:
        if not settings.telegram_bot_token:
            raise ValueError("TELEGRAM_BOT_TOKEN not configured")
        _app = (
            Application.builder()
            .token(settings.telegram_bot_token)
            .build()
        )
        _register_handlers(_app)
    return _app


def _register_handlers(app: Application):
    app.add_handler(CommandHandler("start", _cmd_start))
    app.add_handler(CommandHandler("help", _cmd_help))
    app.add_handler(CommandHandler("balance", _cmd_balance))
    app.add_handler(CommandHandler("spend", _cmd_spend))
    app.add_handler(CommandHandler("networth", _cmd_networth))
    app.add_handler(CommandHandler("upcoming", _cmd_upcoming))
    app.add_handler(CommandHandler("investments", _cmd_investments))
    app.add_handler(CommandHandler("subscriptions", _cmd_subscriptions))
    app.add_handler(CommandHandler("morning", _cmd_morning))
    app.add_handler(CommandHandler("evening", _cmd_evening))
    app.add_handler(CommandHandler("sync", _cmd_sync))


async def send_report(text: str, chat_id: str = None):
    """Send a report message to the configured Telegram chat."""
    chat_id = chat_id or settings.telegram_chat_id
    if not chat_id:
        logger.warning("No Telegram chat ID configured")
        return

    app = get_bot_app()
    try:
        await app.bot.send_message(
            chat_id=chat_id,
            text=text,
            parse_mode="Markdown",
        )
    except Exception:
        # Retry without markdown if parsing fails
        try:
            await app.bot.send_message(chat_id=chat_id, text=text)
        except Exception:
            logger.exception("Failed to send Telegram message")


async def send_morning_report(chat_id: str = None):
    report = generate_morning_report()
    await send_report(report, chat_id)


async def send_evening_report(chat_id: str = None):
    report = generate_evening_report()
    await send_report(report, chat_id)


# ── Command Handlers ─────────────────────────────────────────────────────

async def _cmd_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text(
        "🏦 *Finance Agent* is active!\n\n"
        "I track your finances and send daily reports.\n"
        "Use /help to see available commands.",
        parse_mode="Markdown",
    )


async def _cmd_help(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text(
        "📋 *Available Commands*\n\n"
        "/balance — All account balances\n"
        "/spend [days] — Spending breakdown (default: 7 days)\n"
        "/networth — Current net worth\n"
        "/upcoming — Upcoming EMIs, premiums, bills\n"
        "/investments — Portfolio summary\n"
        "/subscriptions — Active subscriptions\n"
        "/morning — Full morning report\n"
        "/evening — Full evening report\n"
        "/sync — Trigger manual sync\n",
        parse_mode="Markdown",
    )


async def _cmd_balance(update: Update, context: ContextTypes.DEFAULT_TYPE):
    accounts = [_restore_decimals(a) for a in accounts_col().find({"user_id": "default"})]
    await update.message.reply_text(format_account_summary(accounts), parse_mode="Markdown")


async def _cmd_spend(update: Update, context: ContextTypes.DEFAULT_TYPE):
    days = 7
    if context.args:
        try:
            days = int(context.args[0])
        except ValueError:
            pass

    end = datetime.utcnow()
    start = end - timedelta(days=days)
    spending = aggregate_spending("default", start, end)
    header = f"📊 *Spending — Last {days} Days*\n\n"
    await update.message.reply_text(
        header + format_spending_summary(spending),
        parse_mode="Markdown",
    )


async def _cmd_networth(update: Update, context: ContextTypes.DEFAULT_TYPE):
    latest = net_worth_col().find_one({"user_id": "default"}, sort=[("date", -1)])
    if latest:
        await update.message.reply_text(
            format_net_worth(_restore_decimals(latest)),
            parse_mode="Markdown",
        )
    else:
        await update.message.reply_text("Net worth not calculated yet. Run /sync first.")


async def _cmd_upcoming(update: Update, context: ContextTypes.DEFAULT_TYPE):
    dues = _get_upcoming_dues("default", days=14)
    await update.message.reply_text(format_upcoming_dues(dues), parse_mode="Markdown")


async def _cmd_investments(update: Update, context: ContextTypes.DEFAULT_TYPE):
    investments = list(investments_col().find({"user_id": "default"}))
    if not investments:
        await update.message.reply_text("No investments tracked yet.")
        return

    lines = ["📈 *Investment Portfolio*\n"]
    total_invested = Decimal("0")
    total_current = Decimal("0")

    for inv in investments:
        inv = _restore_decimals(inv)
        invested = inv.get("invested_amount", Decimal("0"))
        current = inv.get("current_value", Decimal("0"))
        total_invested += invested
        total_current += current
        pnl = current - invested
        arrow = "🟢" if pnl >= 0 else "🔴"
        lines.append(
            f"  {arrow} {inv.get('name', 'Unknown')}\n"
            f"    Invested: {format_currency(invested)} → "
            f"Current: {format_currency(current)}"
        )

    total_pnl = total_current - total_invested
    pnl_pct = float(total_pnl / total_invested * 100) if total_invested else 0
    lines.append(f"\n*Total P&L*: {format_currency(total_pnl)} ({pnl_pct:+.2f}%)")
    await update.message.reply_text("\n".join(lines), parse_mode="Markdown")


async def _cmd_subscriptions(update: Update, context: ContextTypes.DEFAULT_TYPE):
    subs = list(subscriptions_col().find({"user_id": "default", "status": "active"}))
    if not subs:
        await update.message.reply_text("No active subscriptions tracked.")
        return

    lines = ["🔄 *Active Subscriptions*\n"]
    total = Decimal("0")
    for sub in subs:
        sub = _restore_decimals(sub)
        amt = sub.get("amount", Decimal("0"))
        total += amt
        lines.append(
            f"  • {sub['name']}: {format_currency(amt)}/{sub.get('frequency', 'month')}"
        )
    lines.append(f"\n*Monthly Total*: ~{format_currency(total)}")
    await update.message.reply_text("\n".join(lines), parse_mode="Markdown")


async def _cmd_morning(update: Update, context: ContextTypes.DEFAULT_TYPE):
    report = generate_morning_report()
    await update.message.reply_text(report, parse_mode="Markdown")


async def _cmd_evening(update: Update, context: ContextTypes.DEFAULT_TYPE):
    report = generate_evening_report()
    await update.message.reply_text(report, parse_mode="Markdown")


async def _cmd_sync(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("🔄 Starting manual sync...")
    try:
        from src.scheduler import run_sync_pipeline
        count = await run_sync_pipeline()
        await update.message.reply_text(f"✅ Sync complete! Processed {count} new transactions.")
    except Exception as e:
        await update.message.reply_text(f"❌ Sync failed: {str(e)}")
