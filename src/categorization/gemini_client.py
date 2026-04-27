"""Gemini API client for categorizing ambiguous transactions.

Called as fallback when the rule-based engine cannot determine category.
Batches transactions for efficiency and caches results per merchant.
"""

import json
import logging
from typing import Optional

from config.settings import settings

logger = logging.getLogger(__name__)

_merchant_cache: dict[str, dict] = {}


CATEGORIZATION_PROMPT = """You are a financial transaction categorizer for an Indian user's bank transactions.

Categorize each transaction into exactly one category and sub-category from this list:

Categories:
- food_dining: restaurants, groceries, food_delivery, cafe, bakery
- shopping: online, clothing, electronics, home_furnishing, accessories, cosmetics
- transport: fuel, cab, auto, parking, toll, metro, bus, train, flight
- bills_utilities: electricity, water, gas, internet, phone, dth, recharge
- entertainment: movies, streaming, gaming, events, sports, music
- health_fitness: pharmacy, hospital, doctor, gym, supplements
- education: courses, books, tuition, school, coaching
- investment: stocks, mutual_funds, fd, rd, gold, crypto, ppf, nps
- emi_loans: home_loan, car_loan, personal_loan, education_loan
- insurance: life, health, vehicle, term
- subscriptions: monthly, yearly, streaming, saas
- transfers: self_transfer, family, friends, salary
- cash_withdrawal: atm
- tax_government: income_tax, gst, challan
- rent: house_rent, office_rent
- travel: hotel, booking
- personal_care: salon, spa, laundry
- charity: donation, temple, ngo
- others: uncategorized

For each transaction, return a JSON array with objects containing:
- "index": the transaction index (0-based)
- "category": one of the categories above
- "sub_category": one of the sub-categories
- "merchant": cleaned-up merchant name (or best guess)
- "tags": array of relevant tags like ["essential", "discretionary", "recurring"]

Respond with ONLY valid JSON, no markdown or explanation.

Transactions to categorize:
{transactions}"""


def categorize_with_gemini(transactions: list[dict]) -> list[dict]:
    """Send uncategorized transactions to Gemini for categorization.

    Returns the transactions with category fields populated.
    Falls back gracefully if Gemini is unavailable.
    """
    if not settings.gemini_api_key:
        logger.warning("Gemini API key not configured, skipping LLM categorization")
        return transactions

    # Check cache first
    uncached = []
    for i, txn in enumerate(transactions):
        merchant = (txn.get("merchant") or "").lower().strip()
        if merchant and merchant in _merchant_cache:
            cached = _merchant_cache[merchant]
            txn["category"] = cached["category"]
            txn["sub_category"] = cached["sub_category"]
            txn["tags"] = cached.get("tags", [])
            txn["categorized_by"] = "gemini"
        else:
            uncached.append((i, txn))

    if not uncached:
        return transactions

    # Batch uncached for Gemini
    batch_descriptions = []
    for idx, (orig_idx, txn) in enumerate(uncached):
        desc = {
            "index": idx,
            "description": txn.get("description", "")[:200],
            "merchant": txn.get("merchant", ""),
            "amount": str(txn.get("amount", "")),
            "payment_mode": txn.get("payment_mode", ""),
        }
        batch_descriptions.append(desc)

    try:
        import google.generativeai as genai
        genai.configure(api_key=settings.gemini_api_key)
        model = genai.GenerativeModel(settings.gemini_model)

        prompt = CATEGORIZATION_PROMPT.format(
            transactions=json.dumps(batch_descriptions, indent=2)
        )
        response = model.generate_content(prompt)
        text = response.text.strip()

        # Strip markdown code fences if present
        if text.startswith("```"):
            text = text.split("\n", 1)[1] if "\n" in text else text[3:]
        if text.endswith("```"):
            text = text[:-3]
        text = text.strip()

        results = json.loads(text)

        for result in results:
            idx = result.get("index", -1)
            if 0 <= idx < len(uncached):
                orig_idx, txn = uncached[idx]
                txn["category"] = result.get("category", "others")
                txn["sub_category"] = result.get("sub_category", "uncategorized")
                txn["merchant"] = result.get("merchant", txn.get("merchant", ""))
                txn["tags"] = result.get("tags", [])
                txn["categorized_by"] = "gemini"

                # Cache by merchant
                merchant_key = txn["merchant"].lower().strip()
                if merchant_key:
                    _merchant_cache[merchant_key] = {
                        "category": txn["category"],
                        "sub_category": txn["sub_category"],
                        "tags": txn["tags"],
                    }

        logger.info("Gemini categorized %d/%d transactions", len(results), len(uncached))

    except json.JSONDecodeError:
        logger.error("Gemini returned invalid JSON, skipping categorization")
    except Exception:
        logger.exception("Gemini categorization failed")

    return transactions


def clear_cache():
    _merchant_cache.clear()
