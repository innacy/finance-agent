"""Parse CAMS/KFintech Consolidated Account Statement (CAS) for mutual fund holdings.

CAS PDFs contain the complete mutual fund portfolio across all AMCs.
Available for free from: https://www.camsonline.com/Investors/Statements/Consolidated-Account-Statement
"""

import re
import logging
from datetime import datetime
from decimal import Decimal, InvalidOperation
from typing import Optional

import pdfplumber

logger = logging.getLogger(__name__)


class CASParser:
    """Parse CAMS/KFintech Consolidated Account Statement PDF."""

    def parse(self, file_path: str) -> dict:
        """Parse CAS PDF and return portfolio data.

        Returns: {
            "investor_name": str,
            "pan": str,
            "as_on_date": datetime,
            "folios": [{
                "amc": str,
                "folio": str,
                "schemes": [{
                    "name": str,
                    "isin": str,
                    "units": Decimal,
                    "nav": Decimal,
                    "value": Decimal,
                    "transactions": [{date, type, units, amount, nav}]
                }]
            }],
            "total_value": Decimal,
        }
        """
        result = {
            "investor_name": "",
            "pan": "",
            "as_on_date": None,
            "folios": [],
            "total_value": Decimal("0"),
        }

        try:
            with pdfplumber.open(file_path) as pdf:
                full_text = ""
                for page in pdf.pages:
                    full_text += page.extract_text() or ""
                    full_text += "\n---PAGE---\n"

                result["investor_name"] = self._extract_investor_name(full_text)
                result["pan"] = self._extract_pan(full_text)
                result["as_on_date"] = self._extract_date(full_text)
                result["folios"] = self._extract_folios(full_text)
                result["total_value"] = sum(
                    s.get("value", Decimal("0"))
                    for f in result["folios"]
                    for s in f.get("schemes", [])
                )

        except Exception:
            logger.exception("Failed to parse CAS file: %s", file_path)

        return result

    def parse_bytes(self, data: bytes) -> dict:
        """Parse CAS PDF from bytes."""
        import io
        result = {
            "investor_name": "", "pan": "", "as_on_date": None,
            "folios": [], "total_value": Decimal("0"),
        }
        try:
            with pdfplumber.open(io.BytesIO(data)) as pdf:
                full_text = ""
                for page in pdf.pages:
                    full_text += page.extract_text() or ""
                    full_text += "\n---PAGE---\n"

                result["investor_name"] = self._extract_investor_name(full_text)
                result["pan"] = self._extract_pan(full_text)
                result["as_on_date"] = self._extract_date(full_text)
                result["folios"] = self._extract_folios(full_text)
                result["total_value"] = sum(
                    s.get("value", Decimal("0"))
                    for f in result["folios"]
                    for s in f.get("schemes", [])
                )
        except Exception:
            logger.exception("Failed to parse CAS bytes")
        return result

    def _extract_investor_name(self, text: str) -> str:
        m = re.search(r"(?:Name|Investor)\s*:\s*([A-Z\s]+?)(?:\n|PAN)", text, re.IGNORECASE)
        return m.group(1).strip() if m else ""

    def _extract_pan(self, text: str) -> str:
        m = re.search(r"PAN\s*:\s*([A-Z]{5}\d{4}[A-Z])", text)
        return m.group(1) if m else ""

    def _extract_date(self, text: str) -> Optional[datetime]:
        m = re.search(r"(?:as on|statement period.*?to)\s*(\d{1,2}[-/]\w{3}[-/]\d{4})", text, re.IGNORECASE)
        if m:
            date_str = m.group(1).replace("/", "-")
            for fmt in ["%d-%b-%Y", "%d-%B-%Y"]:
                try:
                    return datetime.strptime(date_str, fmt)
                except ValueError:
                    continue
        return None

    def _extract_folios(self, text: str) -> list[dict]:
        folios = []
        # Split by AMC sections
        amc_pattern = re.compile(
            r"([\w\s]+(?:Mutual Fund|Asset Management|AMC)[\w\s]*)\n"
            r".*?Folio No:\s*(\S+)",
            re.IGNORECASE | re.DOTALL,
        )

        # Simplified: look for scheme blocks
        scheme_pattern = re.compile(
            r"([A-Z][\w\s\-()]+(?:Fund|Plan|Growth|Dividend|IDCW|Direct)[\w\s\-()]*)\s*"
            r"(?:.*?ISIN:\s*([A-Z0-9]+))?"
            r".*?"
            r"(?:Closing Unit Balance|Valuation on)\s*:?\s*([\d,.]+)\s+"
            r"(?:NAV on.*?:?\s*([\d,.]+)\s+)?"
            r"(?:Valuation on.*?:?\s*(?:INR\s*)?([\d,.]+))?",
            re.IGNORECASE | re.DOTALL,
        )

        # Alternative simpler pattern for holdings summary
        holding_pattern = re.compile(
            r"([A-Z][\w\s\-()]+(?:Fund|Plan|Growth|Dividend|IDCW|Direct)[\w\s\-()]*)\n"
            r".*?"
            r"([\d,]+\.?\d*)\s+(?:units?)?\s+"
            r"(?:[\d,]+\.?\d*)\s+"
            r"([\d,]+\.?\d*)",
            re.IGNORECASE | re.DOTALL,
        )

        current_folio = {"amc": "", "folio": "", "schemes": []}

        for match in holding_pattern.finditer(text):
            scheme_name = match.group(1).strip()
            units = self._parse_decimal(match.group(2))
            value = self._parse_decimal(match.group(3))

            if units and value:
                nav = value / units if units > 0 else Decimal("0")
                scheme = {
                    "name": scheme_name,
                    "isin": "",
                    "units": units,
                    "nav": nav,
                    "value": value,
                    "transactions": [],
                }
                current_folio["schemes"].append(scheme)

        if current_folio["schemes"]:
            folios.append(current_folio)

        return folios

    @staticmethod
    def _parse_decimal(val: str) -> Optional[Decimal]:
        if not val:
            return None
        try:
            return Decimal(val.replace(",", ""))
        except InvalidOperation:
            return None


def import_cas_to_investments(cas_data: dict, user_id: str = "default",
                               broker: str = "cas") -> int:
    """Import CAS parsed data into the investments collection."""
    from src.trackers.investment_tracker import add_investment

    count = 0
    for folio in cas_data.get("folios", []):
        for scheme in folio.get("schemes", []):
            name = scheme.get("name", "")
            units = scheme.get("units", Decimal("0"))
            nav = scheme.get("nav", Decimal("0"))
            value = scheme.get("value", Decimal("0"))
            isin = scheme.get("isin", "")

            if units > 0:
                add_investment(
                    user_id=user_id,
                    inv_type="mutual_fund",
                    name=name,
                    symbol=isin,
                    broker=broker,
                    units=units,
                    avg_buy_price=nav,
                    invested_amount=value,
                )
                count += 1

    logger.info("Imported %d mutual fund schemes from CAS", count)
    return count
