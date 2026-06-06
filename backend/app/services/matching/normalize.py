"""Text normalization shared by the symbol loader, matcher, and search."""
import re
import unicodedata

# Corporate suffix tokens stripped to derive a company's "core" name.
CORP_SUFFIXES: set[str] = {
    "inc", "incorporated", "corp", "corporation", "co", "company",
    "ltd", "limited", "llc", "lp", "plc", "sa", "ag", "nv", "se",
    "holdings", "holding", "group", "groups", "the", "and",
    "class", "common", "stock", "shares", "trust", "fund",
}

_non_alnum = re.compile(r"[^a-z0-9]+")
_ws = re.compile(r"\s+")


def normalize_text(text: str) -> str:
    """Lowercase, strip accents/punctuation, collapse whitespace."""
    if not text:
        return ""
    text = unicodedata.normalize("NFKD", text)
    text = text.encode("ascii", "ignore").decode("ascii")
    text = text.lower()
    text = _non_alnum.sub(" ", text)
    return _ws.sub(" ", text).strip()


def strip_suffix(normalized_name: str) -> str:
    """Drop trailing corporate-suffix tokens from an already-normalized name."""
    tokens = normalized_name.split()
    while tokens and tokens[-1] in CORP_SUFFIXES:
        tokens.pop()
    return " ".join(tokens)


def company_name_forms(name: str) -> tuple[str, str]:
    """Return (full_norm, core_norm) for a raw company name.

    full_norm keeps the corporate suffix ("apple inc"); core_norm strips it
    ("apple"). Falls back to full when stripping would empty the name.
    """
    full = normalize_text(name)
    core = strip_suffix(full)
    if not core:
        core = full
    return full, core


def contains_phrase(haystack_norm: str, phrase_norm: str) -> bool:
    """Whole-phrase (word-boundary) containment in normalized text."""
    if not phrase_norm:
        return False
    return f" {phrase_norm} " in f" {haystack_norm} "
