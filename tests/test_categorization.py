"""Tests for rule-based transaction categorization."""

from decimal import Decimal
from datetime import datetime

from src.categorization.rules_engine import categorize_transaction, categorize_batch


class TestRulesEngine:
    def test_swiggy_upi(self):
        txn = {
            "merchant": "",
            "upi_id": "swiggy@ybl",
            "description": "UPI to swiggy@ybl",
            "amount": Decimal("450"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "food_dining"
        assert result["sub_category"] == "food_delivery"
        assert result["merchant"] == "Swiggy"
        assert result["categorized_by"] == "rules"

    def test_uber_merchant(self):
        txn = {
            "merchant": "Uber India",
            "description": "POS UBER INDIA BANGALORE",
            "amount": Decimal("250"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "transport"
        assert result["sub_category"] == "cab"

    def test_amazon_shopping(self):
        txn = {
            "merchant": "Amazon",
            "description": "Amazon Pay purchase",
            "amount": Decimal("3500"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "shopping"
        assert result["sub_category"] == "online"

    def test_salary_credit(self):
        txn = {
            "merchant": "",
            "description": "NEFT-SALARY-COMPANY NAME",
            "amount": Decimal("50000"),
            "type": "credit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "transfers"
        assert result["sub_category"] == "salary"

    def test_atm_withdrawal(self):
        txn = {
            "merchant": "",
            "description": "ATM-WDL-HDFC BANK ATM",
            "amount": Decimal("5000"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "cash_withdrawal"

    def test_petrol(self):
        txn = {
            "merchant": "Indian Oil",
            "description": "POS INDIAN OIL CORP",
            "amount": Decimal("2000"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["category"] == "transport"
        assert result["sub_category"] == "fuel"

    def test_uncategorized(self):
        txn = {
            "merchant": "",
            "description": "XYZRANDOMMERCHANT12345",
            "amount": Decimal("100"),
            "type": "debit",
        }
        result = categorize_transaction(txn)
        assert result["categorized_by"] == "uncategorized"
        assert result["category"] == "others"

    def test_emi_flag(self):
        txn = {
            "merchant": "",
            "description": "AUTO DEBIT EMI",
            "amount": Decimal("15000"),
            "type": "debit",
            "is_emi": True,
        }
        result = categorize_transaction(txn)
        assert result["category"] == "emi_loans"

    def test_batch_categorization(self):
        txns = [
            {"merchant": "Swiggy", "description": "Swiggy order", "amount": Decimal("300"), "type": "debit"},
            {"merchant": "", "description": "UNKNOWN XYZ", "amount": Decimal("50"), "type": "debit"},
            {"merchant": "Netflix", "description": "Netflix sub", "amount": Decimal("649"), "type": "debit"},
        ]
        categorized, uncategorized = categorize_batch(txns)
        assert len(categorized) == 2
        assert len(uncategorized) == 1
