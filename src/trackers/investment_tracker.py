"""Track stock and mutual fund investments with portfolio P&L."""

import logging
from datetime import datetime
from decimal import Decimal
from typing import Optional

from src.db.mongo import investments_col, _restore_decimals, _convert_decimals

logger = logging.getLogger(__name__)


def add_investment(user_id: str, inv_type: str, name: str,
                    symbol: str = "", broker: str = "",
                    units: Decimal = Decimal("0"),
                    avg_buy_price: Decimal = Decimal("0"),
                    invested_amount: Optional[Decimal] = None) -> str:
    """Add or update an investment holding."""
    if invested_amount is None:
        invested_amount = units * avg_buy_price

    existing = investments_col().find_one({
        "user_id": user_id, "symbol": symbol, "broker": broker,
    }) if symbol else None

    if existing:
        existing = _restore_decimals(existing)
        old_units = existing.get("units", Decimal("0"))
        old_invested = existing.get("invested_amount", Decimal("0"))
        new_units = old_units + units
        new_invested = old_invested + invested_amount
        new_avg = new_invested / new_units if new_units > 0 else Decimal("0")

        investments_col().update_one(
            {"_id": existing["_id"]},
            {"$set": _convert_decimals({
                "units": new_units,
                "avg_buy_price": new_avg,
                "invested_amount": new_invested,
                "last_updated": datetime.utcnow(),
            })},
        )
        return str(existing["_id"])

    doc = _convert_decimals({
        "user_id": user_id,
        "type": inv_type,
        "name": name,
        "symbol": symbol,
        "units": units,
        "avg_buy_price": avg_buy_price,
        "current_price": Decimal("0"),
        "current_value": Decimal("0"),
        "invested_amount": invested_amount,
        "pnl": Decimal("0"),
        "pnl_percent": 0.0,
        "broker": broker,
        "last_updated": datetime.utcnow(),
        "transactions": [],
        "created_at": datetime.utcnow(),
    })
    result = investments_col().insert_one(doc)
    return str(result.inserted_id)


def record_trade(user_id: str, symbol: str, broker: str,
                  trade_type: str, units: Decimal, price: Decimal):
    """Record a buy/sell trade and update holdings."""
    inv = investments_col().find_one({
        "user_id": user_id, "symbol": symbol, "broker": broker,
    })

    trade_entry = _convert_decimals({
        "date": datetime.utcnow(),
        "type": trade_type,
        "units": units,
        "price": price,
    })

    if inv:
        inv = _restore_decimals(inv)
        old_units = inv.get("units", Decimal("0"))
        old_invested = inv.get("invested_amount", Decimal("0"))

        if trade_type == "buy":
            new_units = old_units + units
            new_invested = old_invested + (units * price)
        else:  # sell
            new_units = max(old_units - units, Decimal("0"))
            proportion = units / old_units if old_units > 0 else Decimal("0")
            sold_cost = old_invested * proportion
            new_invested = old_invested - sold_cost

        new_avg = new_invested / new_units if new_units > 0 else Decimal("0")

        investments_col().update_one(
            {"_id": inv["_id"]},
            {
                "$set": _convert_decimals({
                    "units": new_units,
                    "avg_buy_price": new_avg,
                    "invested_amount": new_invested,
                    "last_updated": datetime.utcnow(),
                }),
                "$push": {"transactions": trade_entry},
            },
        )
    else:
        if trade_type == "sell":
            logger.warning("Sell for unknown holding %s, creating with negative", symbol)

        add_investment(
            user_id, "stock", symbol, symbol=symbol, broker=broker,
            units=units if trade_type == "buy" else -units,
            avg_buy_price=price,
        )
        investments_col().update_one(
            {"user_id": user_id, "symbol": symbol, "broker": broker},
            {"$push": {"transactions": trade_entry}},
        )


def update_market_prices(user_id: str = "default"):
    """Update current market prices for all holdings.

    Uses free APIs (no broker key required):
    - NSE data for stocks (via unofficial endpoints)
    - AMFI NAV for mutual funds
    """
    for inv in investments_col().find({"user_id": user_id}):
        inv = _restore_decimals(inv)
        symbol = inv.get("symbol", "")
        inv_type = inv.get("type", "")

        price = None
        if inv_type == "stock" and symbol:
            price = _fetch_stock_price(symbol)
        elif inv_type == "mutual_fund" and symbol:
            price = _fetch_mf_nav(symbol)

        if price and price > 0:
            units = inv.get("units", Decimal("0"))
            current_value = units * price
            invested = inv.get("invested_amount", Decimal("0"))
            pnl = current_value - invested
            pnl_pct = float(pnl / invested * 100) if invested > 0 else 0

            investments_col().update_one(
                {"_id": inv["_id"]},
                {"$set": _convert_decimals({
                    "current_price": price,
                    "current_value": current_value,
                    "pnl": pnl,
                    "pnl_percent": pnl_pct,
                    "last_updated": datetime.utcnow(),
                })},
            )


def get_portfolio_summary(user_id: str = "default") -> dict:
    """Get aggregated portfolio summary."""
    total_invested = Decimal("0")
    total_current = Decimal("0")
    holdings = []

    for inv in investments_col().find({"user_id": user_id}):
        inv = _restore_decimals(inv)
        invested = inv.get("invested_amount", Decimal("0"))
        current = inv.get("current_value", Decimal("0"))
        total_invested += invested
        total_current += current
        holdings.append({
            "name": inv.get("name", ""),
            "type": inv.get("type", ""),
            "invested": invested,
            "current": current,
            "pnl": current - invested,
        })

    total_pnl = total_current - total_invested
    return {
        "total_invested": total_invested,
        "total_current_value": total_current,
        "total_pnl": total_pnl,
        "pnl_percent": float(total_pnl / total_invested * 100) if total_invested else 0,
        "holdings_count": len(holdings),
        "holdings": holdings,
    }


def _fetch_stock_price(symbol: str) -> Optional[Decimal]:
    """Fetch stock price from free NSE data source."""
    try:
        import httpx
        url = f"https://www.google.com/finance/quote/{symbol}:NSE"
        resp = httpx.get(url, timeout=10, follow_redirects=True,
                         headers={"User-Agent": "Mozilla/5.0"})
        if resp.status_code == 200:
            import re
            match = re.search(r'data-last-price="([\d,.]+)"', resp.text)
            if match:
                return Decimal(match.group(1).replace(",", ""))
    except Exception:
        logger.debug("Failed to fetch stock price for %s", symbol)
    return None


def _fetch_mf_nav(isin_or_code: str) -> Optional[Decimal]:
    """Fetch mutual fund NAV from AMFI (free, official)."""
    try:
        import httpx
        resp = httpx.get("https://www.amfiindia.com/spages/NAVAll.txt",
                         timeout=15)
        if resp.status_code == 200:
            for line in resp.text.split("\n"):
                parts = line.split(";")
                if len(parts) >= 5:
                    if isin_or_code in line:
                        nav_str = parts[4].strip()
                        try:
                            return Decimal(nav_str)
                        except Exception:
                            pass
    except Exception:
        logger.debug("Failed to fetch MF NAV for %s", isin_or_code)
    return None
