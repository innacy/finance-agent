"""PDF/CSV bank statement ingestion and parsing."""

import csv
import io
import logging
from decimal import Decimal, InvalidOperation
from datetime import datetime
from pathlib import Path
from typing import Optional

import pdfplumber

logger = logging.getLogger(__name__)


class StatementParser:
    """Parse bank statements (PDF and CSV) into normalized transaction dicts."""

    def parse_file(self, file_path: str, bank_name: str = "hdfc") -> list[dict]:
        path = Path(file_path)
        if path.suffix.lower() == ".pdf":
            return self.parse_pdf(file_path, bank_name)
        elif path.suffix.lower() == ".csv":
            return self.parse_csv(file_path, bank_name)
        else:
            raise ValueError(f"Unsupported file type: {path.suffix}")

    def parse_pdf(self, file_path: str, bank_name: str = "hdfc") -> list[dict]:
        transactions = []
        try:
            with pdfplumber.open(file_path) as pdf:
                for page in pdf.pages:
                    tables = page.extract_tables()
                    for table in tables:
                        rows = self._parse_table_rows(table, bank_name)
                        transactions.extend(rows)
        except Exception:
            logger.exception("Failed to parse PDF: %s", file_path)
        logger.info("Extracted %d transactions from PDF %s", len(transactions), file_path)
        return transactions

    def parse_csv(self, file_path: str, bank_name: str = "hdfc") -> list[dict]:
        transactions = []
        try:
            with open(file_path, "r", encoding="utf-8-sig") as f:
                reader = csv.DictReader(f)
                for row in reader:
                    txn = self._normalize_csv_row(row, bank_name)
                    if txn:
                        transactions.append(txn)
        except Exception:
            logger.exception("Failed to parse CSV: %s", file_path)
        logger.info("Extracted %d transactions from CSV %s", len(transactions), file_path)
        return transactions

    def parse_pdf_bytes(self, data: bytes, bank_name: str = "hdfc") -> list[dict]:
        transactions = []
        try:
            with pdfplumber.open(io.BytesIO(data)) as pdf:
                for page in pdf.pages:
                    tables = page.extract_tables()
                    for table in tables:
                        rows = self._parse_table_rows(table, bank_name)
                        transactions.extend(rows)
        except Exception:
            logger.exception("Failed to parse PDF bytes")
        return transactions

    def _parse_table_rows(self, table: list[list], bank_name: str) -> list[dict]:
        if not table or len(table) < 2:
            return []

        header = [str(c).strip().lower() if c else "" for c in table[0]]
        date_col = self._find_col(header, ["date", "txn date", "transaction date", "value date"])
        desc_col = self._find_col(header, ["narration", "description", "particulars", "details", "transaction details"])
        debit_col = self._find_col(header, ["debit", "withdrawal", "dr", "debit amount"])
        credit_col = self._find_col(header, ["credit", "deposit", "cr", "credit amount"])
        balance_col = self._find_col(header, ["balance", "closing balance", "running balance"])

        if date_col is None or desc_col is None:
            return []

        results = []
        for row in table[1:]:
            if not row or len(row) <= max(filter(None, [date_col, desc_col, debit_col, credit_col]), default=0):
                continue

            date = self._parse_date(str(row[date_col] or "").strip())
            if not date:
                continue

            description = str(row[desc_col] or "").strip()
            if not description:
                continue

            debit_amt = self._parse_amount(row[debit_col]) if debit_col is not None else None
            credit_amt = self._parse_amount(row[credit_col]) if credit_col is not None else None
            balance = self._parse_amount(row[balance_col]) if balance_col is not None else None

            if debit_amt and debit_amt > 0:
                txn_type = "debit"
                amount = debit_amt
            elif credit_amt and credit_amt > 0:
                txn_type = "credit"
                amount = credit_amt
            else:
                continue

            results.append({
                "date": date,
                "description": description,
                "amount": amount,
                "type": txn_type,
                "balance": balance,
                "source": "statement",
                "bank_name": bank_name.upper(),
            })

        return results

    def _normalize_csv_row(self, row: dict, bank_name: str) -> Optional[dict]:
        norm = {k.strip().lower(): v.strip() if v else "" for k, v in row.items()}

        date_str = norm.get("date") or norm.get("txn date") or norm.get("transaction date") or ""
        date = self._parse_date(date_str)
        if not date:
            return None

        desc = norm.get("narration") or norm.get("description") or norm.get("particulars") or ""
        if not desc:
            return None

        debit = self._parse_amount(norm.get("debit") or norm.get("withdrawal") or "")
        credit = self._parse_amount(norm.get("credit") or norm.get("deposit") or "")

        if debit and debit > 0:
            return {"date": date, "description": desc, "amount": debit, "type": "debit",
                    "source": "statement", "bank_name": bank_name.upper()}
        elif credit and credit > 0:
            return {"date": date, "description": desc, "amount": credit, "type": "credit",
                    "source": "statement", "bank_name": bank_name.upper()}
        return None

    @staticmethod
    def _find_col(header: list[str], candidates: list[str]) -> Optional[int]:
        for i, h in enumerate(header):
            for c in candidates:
                if c in h:
                    return i
        return None

    @staticmethod
    def _parse_date(s: str) -> Optional[datetime]:
        formats = [
            "%d/%m/%Y", "%d-%m-%Y", "%d/%m/%y", "%d-%m-%y",
            "%Y-%m-%d", "%d %b %Y", "%d %b %y", "%d %B %Y",
            "%m/%d/%Y", "%Y/%m/%d",
        ]
        s = s.strip().replace("\n", " ")
        for fmt in formats:
            try:
                return datetime.strptime(s, fmt)
            except ValueError:
                continue
        return None

    @staticmethod
    def _parse_amount(val) -> Optional[Decimal]:
        if val is None:
            return None
        s = str(val).strip().replace(",", "").replace("₹", "").replace("INR", "").replace(" ", "")
        if not s or s == "-" or s.lower() == "nan":
            return None
        try:
            d = Decimal(s)
            return abs(d)
        except InvalidOperation:
            return None
