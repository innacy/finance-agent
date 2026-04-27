"""API response schemas."""

from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, Field


class AccountResponse(BaseModel):
    bank_name: str
    account_number_masked: str
    account_type: str
    current_balance: float
    last_synced: Optional[datetime] = None


class TransactionResponse(BaseModel):
    date: datetime
    amount: float
    type: str
    category: str
    sub_category: str = ""
    merchant: str = ""
    description: str = ""
    payment_mode: str = ""
    tags: list[str] = []
    source: str = ""


class SpendingCategory(BaseModel):
    category: str
    display_name: str
    total: float
    count: int


class SpendingSummary(BaseModel):
    period_start: datetime
    period_end: datetime
    total_spending: float
    categories: list[SpendingCategory]


class NetWorthResponse(BaseModel):
    date: datetime
    total_assets: float
    total_liabilities: float
    net_worth: float
    breakdown: dict


class InvestmentResponse(BaseModel):
    type: str
    name: str
    symbol: str = ""
    units: float = 0
    avg_buy_price: float = 0
    current_value: float = 0
    invested_amount: float = 0
    pnl: float = 0
    pnl_percent: float = 0
    broker: str = ""


class SubscriptionResponse(BaseModel):
    name: str
    amount: float = 0
    frequency: str = "monthly"
    next_billing_date: Optional[datetime] = None
    category: str = ""
    status: str = "active"


class LoanResponse(BaseModel):
    type: str
    lender: str
    outstanding: float = 0
    emi_amount: float = 0
    interest_rate: float = 0
    remaining_emis: int = 0
    next_due_date: Optional[datetime] = None


class InsuranceResponse(BaseModel):
    type: str
    provider: str
    premium_amount: float = 0
    premium_frequency: str = "yearly"
    next_premium_date: Optional[datetime] = None
    sum_assured: float = 0
    status: str = "active"


class SyncResponse(BaseModel):
    status: str
    transactions_processed: int
    message: str


class UploadResponse(BaseModel):
    status: str
    transactions_found: int
    transactions_stored: int
    duplicates_skipped: int
