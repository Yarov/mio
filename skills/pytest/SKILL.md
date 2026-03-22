---
name: pytest
description: >
  Pytest testing patterns for Python.
  Trigger: When writing Python tests - fixtures, mocking, markers.
metadata:
  author: mio
  version: "1.0"
---

## Basic Test Structure

```python
import pytest

class TestUserService:
    def test_create_user_success(self):
        user = create_user(name="John", email="john@test.com")
        assert user.name == "John"
        assert user.email == "john@test.com"

    def test_create_user_invalid_email_fails(self):
        with pytest.raises(ValueError, match="Invalid email"):
            create_user(name="John", email="invalid")
```

## Fixtures

```python
@pytest.fixture
def user():
    return User(name="Test User", email="test@example.com")

@pytest.fixture
def authenticated_client(client, user):
    client.force_login(user)
    return client

# Fixture with teardown
@pytest.fixture
def temp_file():
    path = Path("/tmp/test_file.txt")
    path.write_text("test content")
    yield path
    path.unlink()

# Scopes: function (default), class, module, session
@pytest.fixture(scope="session")
def db_connection():
    conn = create_connection()
    yield conn
    conn.close()
```

## conftest.py

```python
# tests/conftest.py - Shared fixtures (auto-discovered)
@pytest.fixture
def db_session():
    session = create_session()
    yield session
    session.rollback()
```

## Mocking

```python
from unittest.mock import patch, MagicMock

class TestPaymentService:
    def test_process_payment_success(self):
        with patch("services.payment.stripe_client") as mock_stripe:
            mock_stripe.charge.return_value = {"status": "succeeded"}
            result = process_payment(amount=100)
            assert result["status"] == "succeeded"
            mock_stripe.charge.assert_called_once_with(amount=100)

    def test_process_payment_failure(self):
        with patch("services.payment.stripe_client") as mock_stripe:
            mock_stripe.charge.side_effect = PaymentError("Card declined")
            with pytest.raises(PaymentError):
                process_payment(amount=100)
```

## Parametrize

```python
@pytest.mark.parametrize("email,is_valid", [
    ("user@example.com", True),
    ("invalid-email", False),
    ("", False),
    ("user@.com", False),
])
def test_email_validation(email, is_valid):
    assert validate_email(email) == is_valid
```

## Markers

```python
@pytest.mark.slow
def test_large_data_processing(): ...

@pytest.mark.integration
def test_database_connection(): ...

@pytest.mark.skip(reason="Not implemented yet")
def test_future_feature(): ...

@pytest.mark.asyncio
async def test_async_function():
    result = await async_fetch_data()
    assert result is not None
```

## Commands

```bash
pytest                          # Run all tests
pytest -v                       # Verbose output
pytest -x                       # Stop on first failure
pytest -k "test_user"           # Filter by name
pytest -m "not slow"            # Filter by marker
pytest --cov=src                # With coverage
pytest -n auto                  # Parallel (pytest-xdist)
```

## Keywords
pytest, python, testing, fixtures, mocking, parametrize, markers
