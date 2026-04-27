"""Bulk import from bank statements and CAS for initial data load.

Usage:
    # Import bank statement
    python -m scripts.initial_import --type statement --file data/hdfc_2024.pdf --bank hdfc --account XX1234

    # Import CAS for mutual funds
    python -m scripts.initial_import --type cas --file data/cas_consolidated.pdf

    # Import all statements from a directory
    python -m scripts.initial_import --type statement --dir data/statements/ --bank hdfc --account XX1234
"""

import argparse
import logging
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from config.settings import settings
from src.db.mongo import ensure_indexes, insert_transaction, compute_dedup_hash
from src.sources.statement_parser import StatementParser
from src.parsers.statement_extractor import normalize_statement_transactions
from src.parsers.cas_parser import CASParser, import_cas_to_investments
from src.categorization.rules_engine import categorize_transaction
from src.categorization.gemini_client import categorize_with_gemini

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)


def import_statement(file_path: str, bank_name: str, account_masked: str,
                      user_id: str = "default") -> dict:
    """Import a single bank statement file."""
    logger.info("Importing statement: %s (bank=%s, account=%s)",
                file_path, bank_name, account_masked)

    parser = StatementParser()
    raw_txns = parser.parse_file(file_path, bank_name)
    logger.info("Extracted %d raw transactions", len(raw_txns))

    normalized = normalize_statement_transactions(raw_txns, bank_name.upper(), account_masked)
    logger.info("Normalized %d transactions", len(normalized))

    stored = 0
    duplicates = 0
    uncategorized_list = []

    for txn in normalized:
        txn = categorize_transaction(txn)
        txn["user_id"] = user_id

        from decimal import Decimal
        if not txn.get("dedup_hash"):
            txn["dedup_hash"] = compute_dedup_hash(
                txn["date"], txn.get("amount", Decimal("0")),
                txn.get("description", ""), txn.get("account_number_masked", ""),
            )

        result = insert_transaction(txn)
        if result:
            stored += 1
            if txn.get("categorized_by") == "uncategorized":
                uncategorized_list.append(txn)
        else:
            duplicates += 1

    # Batch categorize uncategorized via Gemini
    if uncategorized_list:
        logger.info("Sending %d uncategorized to Gemini...", len(uncategorized_list))
        categorize_with_gemini(uncategorized_list)

    stats = {
        "file": file_path,
        "total_extracted": len(raw_txns),
        "stored": stored,
        "duplicates": duplicates,
        "uncategorized": len(uncategorized_list),
    }
    logger.info("Import complete: %s", stats)
    return stats


def import_cas(file_path: str, user_id: str = "default") -> dict:
    """Import CAMS/KFintech CAS for mutual fund holdings."""
    logger.info("Importing CAS: %s", file_path)

    parser = CASParser()
    data = parser.parse(file_path)

    logger.info("CAS: Investor=%s, PAN=%s, Folios=%d",
                data.get("investor_name"), data.get("pan"),
                len(data.get("folios", [])))

    count = import_cas_to_investments(data, user_id)

    stats = {
        "file": file_path,
        "investor": data.get("investor_name", ""),
        "total_value": str(data.get("total_value", 0)),
        "schemes_imported": count,
    }
    logger.info("CAS import complete: %s", stats)
    return stats


def import_directory(dir_path: str, bank_name: str, account_masked: str,
                      user_id: str = "default") -> list[dict]:
    """Import all PDF/CSV files from a directory."""
    results = []
    for filename in sorted(os.listdir(dir_path)):
        if filename.lower().endswith((".pdf", ".csv")):
            file_path = os.path.join(dir_path, filename)
            result = import_statement(file_path, bank_name, account_masked, user_id)
            results.append(result)
    return results


def main():
    parser = argparse.ArgumentParser(description="Bulk import financial data")
    parser.add_argument("--type", choices=["statement", "cas"], required=True,
                        help="Type of import: statement or cas")
    parser.add_argument("--file", help="Path to a single file")
    parser.add_argument("--dir", help="Path to directory of statements")
    parser.add_argument("--bank", default="hdfc", help="Bank name (default: hdfc)")
    parser.add_argument("--account", default="", help="Masked account number (e.g. XX1234)")
    parser.add_argument("--user", default="default", help="User ID")

    args = parser.parse_args()

    ensure_indexes()

    if args.type == "cas":
        if not args.file:
            print("ERROR: --file is required for CAS import")
            sys.exit(1)
        import_cas(args.file, args.user)

    elif args.type == "statement":
        if args.file:
            import_statement(args.file, args.bank, args.account, args.user)
        elif args.dir:
            import_directory(args.dir, args.bank, args.account, args.user)
        else:
            print("ERROR: --file or --dir is required for statement import")
            sys.exit(1)

    print("\nImport complete!")


if __name__ == "__main__":
    main()
