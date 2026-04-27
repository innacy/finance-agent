"""Pydantic models for all MongoDB collections."""

from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


def _now() -> datetime:
    return datetime.utcnow()


# ── Enums ────────────────────────────────────────────────────────────────

class AccountType(str, Enum):
    SAVINGS = "savings"
    CURRENT = "current"
    SALARY = "salary"


class TransactionType(str, Enum):
    DEBIT = "debit"
    CREDIT = "credit"


class PaymentMode(str, Enum):
    UPI = "UPI"
    NEFT = "NEFT"
    IMPS = "IMPS"
    CARD = "card"
    AUTO_DEBIT = "auto_debit"
    CASH = "cash"
    CHEQUE = "cheque"
    RTGS = "RTGS"
    OTHER = "other"


class DataSource(str, Enum):
    EMAIL = "email"
    STATEMENT = "statement"
    MANUAL = "manual"
    SMS = "sms"


class CategorizationMethod(str, Enum):
    RULES = "rules"
    GEMINI = "gemini"
    MANUAL = "manual"
    UNCATEGORIZED = "uncategorized"


class InvestmentType(str, Enum):
    STOCK = "stock"
    MUTUAL_FUND = "mutual_fund"


class FdRdType(str, Enum):
    FD = "fd"
    RD = "rd"


class FdRdStatus(str, Enum):
    ACTIVE = "active"
    MATURED = "matured"
    PREMATURE_CLOSED = "premature_closed"


class LoanType(str, Enum):
    HOME_LOAN = "home_loan"
    CAR_LOAN = "car_loan"
    PERSONAL_LOAN = "personal_loan"
    CREDIT_CARD = "credit_card"
    EDUCATION_LOAN = "education_loan"
    GOLD_LOAN = "gold_loan"


class InsuranceType(str, Enum):
    LIFE = "life"
    HEALTH = "health"
    VEHICLE = "vehicle"
    TERM = "term"


class PremiumFrequency(str, Enum):
    MONTHLY = "monthly"
    QUARTERLY = "quarterly"
    HALF_YEARLY = "half_yearly"
    YEARLY = "yearly"


class PolicyStatus(str, Enum):
    ACTIVE = "active"
    LAPSED = "lapsed"
    EXPIRED = "expired"


class SubscriptionFrequency(str, Enum):
    MONTHLY = "monthly"
    YEARLY = "yearly"
    QUARTERLY = "quarterly"
    WEEKLY = "weekly"


class SubscriptionStatus(str, Enum):
    ACTIVE = "active"
    CANCELLED = "cancelled"
    PAUSED = "paused"


# ── Collection Models ────────────────────────────────────────────────────

class Account(BaseModel):
    user_id: str = "default"
    bank_name: str
    account_number_masked: str
    account_type: AccountType = AccountType.SAVINGS
    current_balance: Decimal = Decimal("0")
    last_synced: datetime = Field(default_factory=_now)
    metadata: dict = Field(default_factory=dict)
    created_at: datetime = Field(default_factory=_now)
    updated_at: datetime = Field(default_factory=_now)


class Transaction(BaseModel):
    user_id: str = "default"
    account_id: Optional[str] = None
    date: datetime
    amount: Decimal
    type: TransactionType
    category: str = "others"
    sub_category: str = ""
    merchant: str = ""
    description: str = ""
    payment_mode: PaymentMode = PaymentMode.OTHER
    tags: list[str] = Field(default_factory=list)
    is_recurring: bool = False
    source: DataSource = DataSource.EMAIL
    raw_data: dict = Field(default_factory=dict)
    categorized_by: CategorizationMethod = CategorizationMethod.UNCATEGORIZED
    dedup_hash: str = ""
    created_at: datetime = Field(default_factory=_now)


class InvestmentTransaction(BaseModel):
    date: datetime
    type: str  # buy / sell
    units: Decimal
    price: Decimal


class Investment(BaseModel):
    user_id: str = "default"
    type: InvestmentType
    name: str
    symbol: str = ""
    units: Decimal = Decimal("0")
    avg_buy_price: Decimal = Decimal("0")
    current_price: Decimal = Decimal("0")
    current_value: Decimal = Decimal("0")
    invested_amount: Decimal = Decimal("0")
    pnl: Decimal = Decimal("0")
    pnl_percent: float = 0.0
    broker: str = ""
    last_updated: datetime = Field(default_factory=_now)
    transactions: list[InvestmentTransaction] = Field(default_factory=list)
    created_at: datetime = Field(default_factory=_now)


class FdRd(BaseModel):
    user_id: str = "default"
    type: FdRdType
    bank_name: str
    principal: Decimal
    interest_rate: float
    maturity_date: datetime
    maturity_amount: Decimal = Decimal("0")
    monthly_installment: Decimal = Decimal("0")
    start_date: Optional[datetime] = None
    status: FdRdStatus = FdRdStatus.ACTIVE
    created_at: datetime = Field(default_factory=_now)


class LoanPayment(BaseModel):
    date: datetime
    amount: Decimal
    principal: Decimal = Decimal("0")
    interest: Decimal = Decimal("0")


class LoanCredit(BaseModel):
    user_id: str = "default"
    type: LoanType
    lender: str
    account_masked: str = ""
    total_amount: Decimal = Decimal("0")
    outstanding: Decimal = Decimal("0")
    emi_amount: Decimal = Decimal("0")
    interest_rate: float = 0.0
    emi_date: int = 1
    tenure_months: int = 0
    remaining_emis: int = 0
    next_due_date: Optional[datetime] = None
    payment_history: list[LoanPayment] = Field(default_factory=list)
    created_at: datetime = Field(default_factory=_now)
    updated_at: datetime = Field(default_factory=_now)


class Insurance(BaseModel):
    user_id: str = "default"
    type: InsuranceType
    provider: str
    policy_number_masked: str = ""
    premium_amount: Decimal = Decimal("0")
    premium_frequency: PremiumFrequency = PremiumFrequency.YEARLY
    next_premium_date: Optional[datetime] = None
    sum_assured: Decimal = Decimal("0")
    status: PolicyStatus = PolicyStatus.ACTIVE
    created_at: datetime = Field(default_factory=_now)


class Subscription(BaseModel):
    user_id: str = "default"
    name: str
    amount: Decimal = Decimal("0")
    frequency: SubscriptionFrequency = SubscriptionFrequency.MONTHLY
    next_billing_date: Optional[datetime] = None
    category: str = ""
    payment_method: str = ""
    auto_detected: bool = False
    status: SubscriptionStatus = SubscriptionStatus.ACTIVE
    created_at: datetime = Field(default_factory=_now)


class NetWorthBreakdown(BaseModel):
    bank_balances: Decimal = Decimal("0")
    investments: Decimal = Decimal("0")
    fd_rd: Decimal = Decimal("0")
    insurance_value: Decimal = Decimal("0")
    loans_outstanding: Decimal = Decimal("0")
    credit_card_dues: Decimal = Decimal("0")


class NetWorthSnapshot(BaseModel):
    user_id: str = "default"
    date: datetime = Field(default_factory=_now)
    total_assets: Decimal = Decimal("0")
    total_liabilities: Decimal = Decimal("0")
    net_worth: Decimal = Decimal("0")
    breakdown: NetWorthBreakdown = Field(default_factory=NetWorthBreakdown)
    created_at: datetime = Field(default_factory=_now)


class SyncState(BaseModel):
    """Tracks incremental sync progress per data source."""
    user_id: str = "default"
    source: str
    last_sync_time: datetime = Field(default_factory=_now)
    last_email_id: Optional[str] = None
    last_history_id: Optional[str] = None
    metadata: dict = Field(default_factory=dict)
    updated_at: datetime = Field(default_factory=_now)
