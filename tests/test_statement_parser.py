"""Tests for statement parser and extractor."""

import csv
import os
import tempfile
from datetime import datetime
from decimal import Decimal

from src.sources.statement_parser import StatementParser
from src.parsers.statement_extractor import normalize_statement_transactions


class TestStatementParser:
    def test_parse_csv(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False, newline="") as f:
            writer = csv.writer(f)
            writer.writerow(["Date", "Narration", "Debit", "Credit", "Balance"])
            writer.writerow(["01/04/2026", "UPI-SWIGGY-swiggy@ybl", "450.00", "", "25000.00"])
            writer.writerow(["02/04/2026", "NEFT-SALARY-COMPANY", "", "50000.00", "75000.00"])
            writer.writerow(["03/04/2026", "ATM-WDL-HDFC ATM", "5000.00", "", "70000.00"])
            f.flush()
            path = f.name

        try:
            parser = StatementParser()
            txns = parser.parse_csv(path, "hdfc")
            assert len(txns) == 3
            assert txns[0]["type"] == "debit"
            assert txns[0]["amount"] == Decimal("450.00")
            assert txns[1]["type"] == "credit"
            assert txns[1]["amount"] == Decimal("50000.00")
        finally:
            os.unlink(path)

    def test_parse_amount_formats(self):
        parser = StatementParser()
        assert parser._parse_amount("1,250.50") == Decimal("1250.50")
        assert parser._parse_amount("₹5,000") == Decimal("5000")
        assert parser._parse_amount("INR 100.00") == Decimal("100.00")
        assert parser._parse_amount("") is None
        assert parser._parse_amount("-") is None
        assert parser._parse_amount(None) is None

    def test_parse_date_formats(self):
        parser = StatementParser()
        assert parser._parse_date("01/04/2026") == datetime(2026, 4, 1)
        assert parser._parse_date("01-04-2026") == datetime(2026, 4, 1)
        assert parser._parse_date("2026-04-01") == datetime(2026, 4, 1)
        assert parser._parse_date("01 Apr 2026") == datetime(2026, 4, 1)
        assert parser._parse_date("invalid") is None


class TestStatementExtractor:
    def test_normalize(self):
        raw = [
            {
                "date": datetime(2026, 4, 1),
                "description": "UPI-SWIGGY-swiggy@ybl-REF123",
                "amount": Decimal("450"),
                "type": "debit",
                "balance": Decimal("25000"),
            },
        ]
        normalized = normalize_statement_transactions(raw, "HDFC", "XX1234")
        assert len(normalized) == 1
        assert normalized[0]["payment_mode"] == "UPI"
        assert normalized[0]["source"] == "statement"
        assert normalized[0]["dedup_hash"]

    def test_detect_payment_mode(self):
        from src.parsers.statement_extractor import _detect_mode_from_description
        assert _detect_mode_from_description("UPI-SWIGGY") == "UPI"
        assert _detect_mode_from_description("NEFT-SALARY") == "NEFT"
        assert _detect_mode_from_description("IMPS-TRANSFER") == "IMPS"
        assert _detect_mode_from_description("POS AMAZON") == "card"
        assert _detect_mode_from_description("ATM-WDL") == "cash"
        assert _detect_mode_from_description("NACH-ACH") == "auto_debit"
