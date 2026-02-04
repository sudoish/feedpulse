"""Validation utilities for FeedPulse.

This module provides validators for URLs, configuration, and feed data.
"""

from typing import Any, Optional
from urllib.parse import urlparse

from .exceptions import URLValidationError, ValidationError


def validate_url(url: str) -> None:
    """Validate a URL for common issues.

    Args:
        url: URL string to validate

    Raises:
        URLValidationError: If the URL is invalid

    Examples:
        >>> validate_url("https://example.com")  # OK
        >>> validate_url("not-a-url")  # Raises URLValidationError
    """
    if not url:
        raise URLValidationError(url, "URL cannot be empty")

    if not isinstance(url, str):
        raise URLValidationError(str(url), "URL must be a string")

    parsed = urlparse(url)

    if not parsed.scheme:
        raise URLValidationError(url, "Missing URL scheme (http/https)")

    if parsed.scheme not in ("http", "https"):
        raise URLValidationError(
            url, f"Unsupported scheme '{parsed.scheme}' (must be http/https)"
        )

    if not parsed.netloc:
        raise URLValidationError(url, "Missing hostname")

    # Check for common typos
    if ".." in url:
        raise URLValidationError(url, "Contains double dots (..)")

    if " " in url:
        raise URLValidationError(url, "Contains spaces")


def validate_timeout(timeout: Any) -> None:
    """Validate a timeout value.

    Args:
        timeout: Timeout value to validate (should be positive integer)

    Raises:
        ValidationError: If timeout is invalid
    """
    if not isinstance(timeout, int):
        raise ValidationError(f"Timeout must be an integer, got {type(timeout).__name__}")

    if timeout <= 0:
        raise ValidationError(f"Timeout must be positive, got {timeout}")

    if timeout > 300:
        raise ValidationError(f"Timeout too large (max 300s), got {timeout}")


def validate_concurrency(max_concurrency: Any) -> None:
    """Validate max_concurrency value.

    Args:
        max_concurrency: Concurrency limit to validate

    Raises:
        ValidationError: If max_concurrency is invalid
    """
    if not isinstance(max_concurrency, int):
        raise ValidationError(
            f"max_concurrency must be an integer, got {type(max_concurrency).__name__}"
        )

    if max_concurrency <= 0:
        raise ValidationError(f"max_concurrency must be positive, got {max_concurrency}")

    if max_concurrency > 100:
        raise ValidationError(
            f"max_concurrency too large (max 100), got {max_concurrency}"
        )


def validate_feed_config(feed: dict[str, Any]) -> None:
    """Validate a feed configuration.

    Args:
        feed: Feed configuration dictionary

    Raises:
        ValidationError: If feed config is invalid
    """
    required_fields = ["url", "feed_type"]

    for field in required_fields:
        if field not in feed:
            raise ValidationError(f"Feed missing required field: {field}")

    validate_url(feed["url"])

    valid_types = ["hackernews", "reddit", "lobsters", "github"]
    if feed["feed_type"] not in valid_types:
        raise ValidationError(
            f"Invalid feed_type '{feed['feed_type']}'. "
            f"Must be one of: {', '.join(valid_types)}"
        )


def validate_config(config: dict[str, Any]) -> None:
    """Validate entire configuration.

    Args:
        config: Configuration dictionary

    Raises:
        ValidationError: If config is invalid
    """
    if "feeds" not in config:
        raise ValidationError("Config missing 'feeds' section")

    if not isinstance(config["feeds"], list):
        raise ValidationError("'feeds' must be a list")

    if not config["feeds"]:
        raise ValidationError("'feeds' list cannot be empty")

    for i, feed in enumerate(config["feeds"]):
        try:
            validate_feed_config(feed)
        except ValidationError as e:
            raise ValidationError(f"Invalid feed at index {i}: {e}")

    # Validate settings if present
    if "settings" in config:
        settings = config["settings"]

        if "timeout" in settings:
            validate_timeout(settings["timeout"])

        if "max_concurrency" in settings:
            validate_concurrency(settings["max_concurrency"])
