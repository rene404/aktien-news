from app.services.matching.normalize import (
    company_name_forms,
    contains_phrase,
    normalize_text,
    strip_suffix,
)


def test_normalize_lowercases_and_strips_punctuation():
    assert normalize_text("Apple Inc.") == "apple inc"
    assert normalize_text("AT&T  Inc!!!") == "at t inc"


def test_normalize_strips_accents():
    assert normalize_text("Nestlé") == "nestle"


def test_strip_suffix_removes_corporate_tokens():
    assert strip_suffix("apple inc") == "apple"
    assert strip_suffix("microsoft corporation") == "microsoft"
    assert strip_suffix("the coca cola company") == "the coca cola"


def test_company_name_forms():
    full, core = company_name_forms("Apple Inc.")
    assert full == "apple inc"
    assert core == "apple"


def test_company_name_forms_all_suffix_falls_back():
    # name that is entirely suffix tokens should not collapse to empty
    full, core = company_name_forms("Holdings Group")
    assert core == full


def test_contains_phrase_word_boundary():
    assert contains_phrase("i bought apple inc today", "apple inc")
    assert contains_phrase("i ate an apple today", "apple")
    assert not contains_phrase("pineapple sales rose", "apple")
    assert not contains_phrase("applesauce is great", "apple")
