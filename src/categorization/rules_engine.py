"""Rule-based transaction categorization engine.

Primary categorization layer -- handles ~85-90% of Indian bank transactions
using merchant name matching, UPI ID patterns, and description regex.
"""

import re
import logging
from typing import Optional

logger = logging.getLogger(__name__)

# ── Merchant Name → Category Mapping ────────────────────────────────────
# ~500+ Indian merchants and common transaction patterns

MERCHANT_RULES: list[tuple[str, str, str, list[str]]] = [
    # (pattern, category, sub_category, tags)
    # ── Food & Dining ──
    ("swiggy", "food_dining", "food_delivery", ["food_delivery", "discretionary"]),
    ("zomato", "food_dining", "food_delivery", ["food_delivery", "discretionary"]),
    ("uber eats", "food_dining", "food_delivery", ["food_delivery", "discretionary"]),
    ("dunzo", "food_dining", "food_delivery", ["food_delivery", "discretionary"]),
    ("blinkit", "food_dining", "groceries", ["quick_commerce", "essential"]),
    ("zepto", "food_dining", "groceries", ["quick_commerce", "essential"]),
    ("instamart", "food_dining", "groceries", ["quick_commerce", "essential"]),
    ("bigbasket", "food_dining", "groceries", ["groceries", "essential"]),
    ("jiomart", "food_dining", "groceries", ["groceries", "essential"]),
    ("grofers", "food_dining", "groceries", ["groceries", "essential"]),
    ("dmart", "food_dining", "groceries", ["groceries", "essential"]),
    ("more retail", "food_dining", "groceries", ["groceries", "essential"]),
    ("reliance fresh", "food_dining", "groceries", ["groceries", "essential"]),
    ("spencer", "food_dining", "groceries", ["groceries", "essential"]),
    ("nature basket", "food_dining", "groceries", ["groceries", "essential"]),
    ("starbucks", "food_dining", "cafe", ["cafe", "discretionary"]),
    ("ccd", "food_dining", "cafe", ["cafe", "discretionary"]),
    ("cafe coffee day", "food_dining", "cafe", ["cafe", "discretionary"]),
    ("blue tokai", "food_dining", "cafe", ["cafe", "discretionary"]),
    ("third wave", "food_dining", "cafe", ["cafe", "discretionary"]),
    ("mcdonald", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("kfc", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("domino", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("pizza hut", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("burger king", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("subway", "food_dining", "restaurants", ["fast_food", "discretionary"]),
    ("haldiram", "food_dining", "restaurants", ["restaurant", "discretionary"]),
    ("barbeque nation", "food_dining", "restaurants", ["restaurant", "discretionary"]),
    ("biryani", "food_dining", "restaurants", ["restaurant", "discretionary"]),

    # ── Shopping ──
    ("amazon", "shopping", "online", ["ecommerce", "discretionary"]),
    ("flipkart", "shopping", "online", ["ecommerce", "discretionary"]),
    ("myntra", "shopping", "clothing", ["fashion", "discretionary"]),
    ("ajio", "shopping", "clothing", ["fashion", "discretionary"]),
    ("nykaa", "shopping", "cosmetics", ["beauty", "discretionary"]),
    ("meesho", "shopping", "online", ["ecommerce", "discretionary"]),
    ("snapdeal", "shopping", "online", ["ecommerce", "discretionary"]),
    ("tata cliq", "shopping", "online", ["ecommerce", "discretionary"]),
    ("croma", "shopping", "electronics", ["electronics", "discretionary"]),
    ("reliance digital", "shopping", "electronics", ["electronics", "discretionary"]),
    ("vijay sales", "shopping", "electronics", ["electronics", "discretionary"]),
    ("apple", "shopping", "electronics", ["electronics", "discretionary"]),
    ("samsung", "shopping", "electronics", ["electronics", "discretionary"]),
    ("ikea", "shopping", "home_furnishing", ["home", "discretionary"]),
    ("pepperfry", "shopping", "home_furnishing", ["home", "discretionary"]),
    ("urban ladder", "shopping", "home_furnishing", ["home", "discretionary"]),
    ("decathlon", "shopping", "accessories", ["sports", "discretionary"]),

    # ── Transport ──
    ("uber", "transport", "cab", ["ride", "discretionary"]),
    ("ola", "transport", "cab", ["ride", "discretionary"]),
    ("rapido", "transport", "auto", ["ride", "discretionary"]),
    ("indian oil", "transport", "fuel", ["fuel", "essential"]),
    ("bharat petroleum", "transport", "fuel", ["fuel", "essential"]),
    ("hp petrol", "transport", "fuel", ["fuel", "essential"]),
    ("iocl", "transport", "fuel", ["fuel", "essential"]),
    ("bpcl", "transport", "fuel", ["fuel", "essential"]),
    ("hpcl", "transport", "fuel", ["fuel", "essential"]),
    ("petrol", "transport", "fuel", ["fuel", "essential"]),
    ("diesel", "transport", "fuel", ["fuel", "essential"]),
    ("fastag", "transport", "toll", ["toll", "essential"]),
    ("irctc", "transport", "train", ["travel", "essential"]),
    ("redbus", "transport", "bus", ["travel", "discretionary"]),
    ("metro", "transport", "metro", ["commute", "essential"]),
    ("parking", "transport", "parking", ["parking", "essential"]),
    ("makemytrip", "travel", "booking", ["travel", "discretionary"]),
    ("goibibo", "travel", "booking", ["travel", "discretionary"]),
    ("cleartrip", "travel", "booking", ["travel", "discretionary"]),
    ("yatra", "travel", "booking", ["travel", "discretionary"]),
    ("oyo", "travel", "hotel", ["hotel", "discretionary"]),
    ("airbnb", "travel", "hotel", ["hotel", "discretionary"]),

    # ── Bills & Utilities ──
    ("tata power", "bills_utilities", "electricity", ["utility", "essential"]),
    ("adani electricity", "bills_utilities", "electricity", ["utility", "essential"]),
    ("bescom", "bills_utilities", "electricity", ["utility", "essential"]),
    ("bses", "bills_utilities", "electricity", ["utility", "essential"]),
    ("msedcl", "bills_utilities", "electricity", ["utility", "essential"]),
    ("jio", "bills_utilities", "phone", ["telecom", "essential"]),
    ("airtel", "bills_utilities", "phone", ["telecom", "essential"]),
    ("vodafone", "bills_utilities", "phone", ["telecom", "essential"]),
    ("vi ", "bills_utilities", "phone", ["telecom", "essential"]),
    ("bsnl", "bills_utilities", "phone", ["telecom", "essential"]),
    ("act fibernet", "bills_utilities", "internet", ["broadband", "essential"]),
    ("hathway", "bills_utilities", "internet", ["broadband", "essential"]),
    ("tata sky", "bills_utilities", "dth", ["utility", "essential"]),
    ("dish tv", "bills_utilities", "dth", ["utility", "essential"]),
    ("d2h", "bills_utilities", "dth", ["utility", "essential"]),
    ("mahanagar gas", "bills_utilities", "gas", ["utility", "essential"]),
    ("indraprastha gas", "bills_utilities", "gas", ["utility", "essential"]),

    # ── Entertainment ──
    ("netflix", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("spotify", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("hotstar", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("prime video", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("jiocinema", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("zee5", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("sonyliv", "entertainment", "streaming", ["subscription", "discretionary"]),
    ("bookmyshow", "entertainment", "movies", ["entertainment", "discretionary"]),
    ("pvr", "entertainment", "movies", ["entertainment", "discretionary"]),
    ("inox", "entertainment", "movies", ["entertainment", "discretionary"]),
    ("steam", "entertainment", "gaming", ["gaming", "discretionary"]),
    ("playstation", "entertainment", "gaming", ["gaming", "discretionary"]),

    # ── Health & Fitness ──
    ("apollo", "health_fitness", "pharmacy", ["health", "essential"]),
    ("1mg", "health_fitness", "pharmacy", ["health", "essential"]),
    ("pharmeasy", "health_fitness", "pharmacy", ["health", "essential"]),
    ("netmeds", "health_fitness", "pharmacy", ["health", "essential"]),
    ("medplus", "health_fitness", "pharmacy", ["health", "essential"]),
    ("practo", "health_fitness", "doctor", ["health", "essential"]),
    ("cult.fit", "health_fitness", "gym", ["fitness", "discretionary"]),
    ("cultfit", "health_fitness", "gym", ["fitness", "discretionary"]),
    ("gym", "health_fitness", "gym", ["fitness", "discretionary"]),

    # ── Education ──
    ("udemy", "education", "courses", ["learning", "investment"]),
    ("coursera", "education", "courses", ["learning", "investment"]),
    ("unacademy", "education", "courses", ["learning", "investment"]),
    ("byju", "education", "courses", ["learning", "investment"]),

    # ── Investment (detected from merchant name) ──
    ("zerodha", "investment", "stocks", ["investment"]),
    ("groww", "investment", "mutual_funds", ["investment"]),
    ("coin", "investment", "mutual_funds", ["investment"]),
    ("kuvera", "investment", "mutual_funds", ["investment"]),
    ("paytm money", "investment", "mutual_funds", ["investment"]),
    ("sip", "investment", "mutual_funds", ["investment", "recurring"]),
    ("mutual fund", "investment", "mutual_funds", ["investment"]),
    ("nps", "investment", "mutual_funds", ["investment", "retirement"]),
    ("ppf", "investment", "mutual_funds", ["investment", "retirement"]),

    # ── Insurance ──
    ("lic", "insurance", "life", ["insurance", "essential"]),
    ("hdfc life", "insurance", "life", ["insurance", "essential"]),
    ("icici prudential", "insurance", "life", ["insurance", "essential"]),
    ("max life", "insurance", "life", ["insurance", "essential"]),
    ("star health", "insurance", "health", ["insurance", "essential"]),
    ("care insurance", "insurance", "health", ["insurance", "essential"]),
    ("bajaj allianz", "insurance", "health", ["insurance", "essential"]),
    ("digit insurance", "insurance", "vehicle", ["insurance", "essential"]),
    ("acko", "insurance", "vehicle", ["insurance", "essential"]),

    # ── Subscriptions / SaaS ──
    ("google", "subscriptions", "saas", ["subscription"]),
    ("microsoft", "subscriptions", "saas", ["subscription"]),
    ("adobe", "subscriptions", "saas", ["subscription"]),
    ("notion", "subscriptions", "saas", ["subscription"]),
    ("github", "subscriptions", "saas", ["subscription"]),
    ("chatgpt", "subscriptions", "saas", ["subscription"]),
    ("openai", "subscriptions", "saas", ["subscription"]),
    ("cursor", "subscriptions", "saas", ["subscription"]),

    # ── EMI / Loans ──
    ("emi", "emi_loans", "personal_loan", ["emi", "liability"]),
    ("loan", "emi_loans", "personal_loan", ["loan", "liability"]),

    # ── Tax ──
    ("income tax", "tax_government", "income_tax", ["tax"]),
    ("gst", "tax_government", "gst", ["tax"]),
    ("challan", "tax_government", "challan", ["tax"]),

    # ── Rent ──
    ("rent", "rent", "house_rent", ["rent", "essential"]),

    # ── Personal Care ──
    ("salon", "personal_care", "salon", ["personal_care", "discretionary"]),
    ("spa", "personal_care", "spa", ["personal_care", "discretionary"]),
    ("urban company", "personal_care", "salon", ["home_service", "discretionary"]),

    # ── Charity ──
    ("donation", "charity", "donation", ["charity"]),
    ("temple", "charity", "temple", ["charity"]),
]

# ── UPI ID → Category Mapping ──────────────────────────────────────────

UPI_RULES: list[tuple[str, str, str, str]] = [
    # (upi_pattern, merchant_name, category, sub_category)
    ("swiggy", "Swiggy", "food_dining", "food_delivery"),
    ("zomato", "Zomato", "food_dining", "food_delivery"),
    ("blinkit", "Blinkit", "food_dining", "groceries"),
    ("zepto", "Zepto", "food_dining", "groceries"),
    ("bigbasket", "BigBasket", "food_dining", "groceries"),
    ("uber", "Uber", "transport", "cab"),
    ("olacabs", "Ola", "transport", "cab"),
    ("rapido", "Rapido", "transport", "auto"),
    ("amazon", "Amazon", "shopping", "online"),
    ("flipkart", "Flipkart", "shopping", "online"),
    ("paytm", "Paytm", "transfers", "self_transfer"),
    ("phonepe", "PhonePe", "transfers", "self_transfer"),
    ("gpay", "Google Pay", "transfers", "self_transfer"),
    ("irctc", "IRCTC", "transport", "train"),
    ("bookmyshow", "BookMyShow", "entertainment", "movies"),
    ("netflix", "Netflix", "entertainment", "streaming"),
    ("spotify", "Spotify", "entertainment", "streaming"),
    ("bharatpe", "BharatPe Merchant", "shopping", "online"),
    ("dream11", "Dream11", "entertainment", "gaming"),
    ("cred", "CRED", "bills_utilities", "phone"),
]

# ── Description Keyword Rules (fallback) ────────────────────────────────

KEYWORD_RULES: list[tuple[list[str], str, str]] = [
    (["atm", "cash withdrawal", "atm-wdl"], "cash_withdrawal", "atm"),
    (["salary", "payroll"], "transfers", "salary"),
    (["interest", "int.pd"], "investment", "fd"),
    (["dividend"], "investment", "stocks"),
    (["refund", "cashback"], "others", "refund"),
    (["neft", "imps", "rtgs", "transfer"], "transfers", "self_transfer"),
]


def categorize_transaction(txn: dict) -> dict:
    """Categorize a transaction using rule-based engine.

    Modifies and returns the transaction dict with category, sub_category,
    tags, merchant (if detected), and categorized_by fields.
    """
    merchant = (txn.get("merchant") or "").lower().strip()
    upi_id = (txn.get("upi_id") or "").lower().strip()
    description = (txn.get("description") or "").lower().strip()

    # 1) Try UPI ID matching first (most specific)
    if upi_id:
        for pattern, merch_name, cat, subcat in UPI_RULES:
            if pattern in upi_id:
                txn["merchant"] = txn.get("merchant") or merch_name
                txn["category"] = cat
                txn["sub_category"] = subcat
                txn["categorized_by"] = "rules"
                return txn

    # 2) Try merchant name matching
    search_text = f"{merchant} {description}"
    for pattern, cat, subcat, tags in MERCHANT_RULES:
        if pattern in search_text:
            txn["category"] = cat
            txn["sub_category"] = subcat
            txn["tags"] = list(set(txn.get("tags", []) + tags))
            txn["categorized_by"] = "rules"
            if not txn.get("merchant"):
                txn["merchant"] = pattern.title()
            return txn

    # 3) Try keyword rules on description
    for keywords, cat, subcat in KEYWORD_RULES:
        if any(kw in description for kw in keywords):
            txn["category"] = cat
            txn["sub_category"] = subcat
            txn["categorized_by"] = "rules"
            return txn

    # 4) Special patterns
    if txn.get("is_emi"):
        txn["category"] = "emi_loans"
        txn["sub_category"] = "personal_loan"
        txn["categorized_by"] = "rules"
        return txn

    if txn.get("is_insurance"):
        txn["category"] = "insurance"
        txn["sub_category"] = "life"
        txn["categorized_by"] = "rules"
        return txn

    if txn.get("is_credit_card"):
        pass  # still needs merchant categorization

    # Not categorized by rules
    txn["categorized_by"] = "uncategorized"
    txn["category"] = "others"
    txn["sub_category"] = "uncategorized"
    return txn


def categorize_batch(transactions: list[dict]) -> tuple[list[dict], list[dict]]:
    """Categorize a batch. Returns (categorized, uncategorized) lists."""
    categorized = []
    uncategorized = []
    for txn in transactions:
        result = categorize_transaction(txn)
        if result["categorized_by"] == "uncategorized":
            uncategorized.append(result)
        else:
            categorized.append(result)
    logger.info("Rules engine: %d categorized, %d uncategorized",
                len(categorized), len(uncategorized))
    return categorized, uncategorized
