"""Tests for email parser -- HDFC alerts, generic bank emails."""

from datetime import datetime
from decimal import Decimal

from src.parsers.templates.hdfc import parse_hdfc_alert
from src.parsers.templates.common import parse_generic_bank_alert
from src.parsers.email_parser import parse_email, _strip_html


class TestHDFCParser:
    def test_debit_upi(self):
        body = (
            "Dear Customer, Rs.450.00 has been debited from your A/C XXXX1234 "
            "on 25-04-2026 via UPI to SWIGGY@ybl. Ref No: 412345678901. "
            "Available balance: Rs.25,000.50"
        )
        result = parse_hdfc_alert("Transaction alert", body, datetime(2026, 4, 25))
        assert result is not None
        assert result["type"] == "debit"
        assert result["amount"] == Decimal("450.00")
        assert result["payment_mode"] == "UPI"
        assert "1234" in result["account_number_masked"]

    def test_credit_neft(self):
        body = (
            "Dear Customer, Rs.50,000.00 has been credited to your A/C XXXX5678 "
            "via NEFT from COMPANY SALARY on 01-04-2026. "
            "Available balance: Rs.75,000.00"
        )
        result = parse_hdfc_alert("Transaction alert", body, datetime(2026, 4, 1))
        assert result is not None
        assert result["type"] == "credit"
        assert result["amount"] == Decimal("50000.00")
        assert result["payment_mode"] == "NEFT"

    def test_credit_card_alert(self):
        body = (
            "Your HDFC Bank Credit Card XX1234 has been used for "
            "Rs.2,500.00 at Amazon on 20-04-2026."
        )
        result = parse_hdfc_alert("Credit Card Transaction", body, datetime(2026, 4, 20))
        assert result is not None
        assert result["type"] == "debit"
        assert result["amount"] == Decimal("2500.00")
        assert result["payment_mode"] == "card"
        assert result.get("is_credit_card") is True

    def test_emi_alert(self):
        body = (
            "Dear Customer, EMI of Rs.15,000.00 has been debited from your "
            "A/C XXXX9876 on 05-04-2026. Ref: EMI/HOME LOAN."
        )
        result = parse_hdfc_alert("Transaction alert", body, datetime(2026, 4, 5))
        assert result is not None
        assert result.get("is_emi") is True
        assert result["amount"] == Decimal("15000.00")

    def test_no_amount(self):
        body = "Dear Customer, your account statement is ready."
        result = parse_hdfc_alert("HDFC Bank", body, datetime.utcnow())
        assert result is None


class TestGenericParser:
    def test_generic_debit(self):
        body = "INR 1,200.00 debited from A/C XX4567 on 15-Apr-2026."
        result = parse_generic_bank_alert("SBI Alert", body, datetime(2026, 4, 15))
        assert result is not None
        assert result["type"] == "debit"
        assert result["amount"] == Decimal("1200.00")

    def test_generic_credit(self):
        body = "Rs.5000 credited to your account XX8901. Balance: Rs.10,000."
        result = parse_generic_bank_alert("Bank Alert", body, datetime(2026, 4, 10))
        assert result is not None
        assert result["type"] == "credit"
        assert result["amount"] == Decimal("5000")


class TestEmailParser:
    def test_parse_email_hdfc(self):
        email_data = {
            "id": "msg123",
            "from": "alerts@hdfcbank.net",
            "subject": "Transaction alert",
            "body_text": "Rs.100 has been debited from your A/C XX1234 via UPI",
            "date": datetime(2026, 4, 25),
        }
        result = parse_email(email_data)
        assert result is not None
        assert result["type"] == "debit"
        assert result["email_id"] == "msg123"
        assert result["dedup_hash"]

    def test_strip_html(self):
        html = "<p>Rs.500 <b>debited</b> from account</p><br/>Balance: Rs.1000"
        text = _strip_html(html)
        assert "Rs.500" in text
        assert "debited" in text
        assert "<p>" not in text
