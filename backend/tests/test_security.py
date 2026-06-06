import jwt
import pytest

from app.core.security import (
    create_access_token,
    create_refresh_token,
    decode_token,
    hash_password,
    verify_password,
)


def test_password_roundtrip():
    h = hash_password("secret123")
    assert h != "secret123"
    assert verify_password("secret123", h)
    assert not verify_password("wrong", h)


def test_access_token_roundtrip():
    token = create_access_token("user-id-1", "admin")
    payload = decode_token(token)
    assert payload["sub"] == "user-id-1"
    assert payload["role"] == "admin"
    assert payload["type"] == "access"


def test_refresh_token_type():
    payload = decode_token(create_refresh_token("user-id-2", "user"))
    assert payload["type"] == "refresh"


def test_tampered_token_rejected():
    with pytest.raises(jwt.InvalidTokenError):
        decode_token("not.a.token")
