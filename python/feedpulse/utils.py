"""Shared utility functions for FeedPulse."""

import hashlib
import json
from datetime import UTC, datetime
from typing import Any, Optional


def generate_id(title: str, url: str, published: Optional[str] = None) -> str:
    """Generate a stable unique ID for a feed item.

    Uses SHA256 hash of title + URL + published timestamp to ensure
    the same item always generates the same ID.

    Args:
        title: Item title
        url: Item URL
        published: Optional publication timestamp

    Returns:
        64-character hex string (SHA256 hash)

    Examples:
        >>> generate_id("Test", "https://example.com")
        'a1b2c3...'  # 64-char hash
    """
    components = f"{title}|{url}|{published or ''}"
    return hashlib.sha256(components.encode()).hexdigest()


def coerce_to_string(value: Any, max_length: Optional[int] = None) -> str:
    """Convert any value to a string safely.

    Handles None, numbers, lists, dicts, and other types gracefully.

    Args:
        value: Value to convert
        max_length: Optional maximum length (truncates if exceeded)

    Returns:
        String representation of the value

    Examples:
        >>> coerce_to_string(None)
        ''
        >>> coerce_to_string(42)
        '42'
        >>> coerce_to_string(['a', 'b'], max_length=3)
        '["a...'
    """
    if value is None:
        return ""

    if isinstance(value, str):
        result = value
    elif isinstance(value, (int, float)):
        result = str(value)
    elif isinstance(value, (list, dict)):
        result = json.dumps(value)
    else:
        result = str(value)

    if max_length is not None and len(result) > max_length:
        return result[:max_length]

    return result


def parse_timestamp(value: Any) -> Optional[str]:
    """Parse various timestamp formats to ISO 8601.

    Supports:
    - Unix timestamps (int/float)
    - ISO 8601 strings
    - None (returns None)

    Args:
        value: Timestamp value (Unix timestamp or ISO string)

    Returns:
        ISO 8601 string or None if parsing fails

    Examples:
        >>> parse_timestamp(1609459200)
        '2021-01-01T00:00:00+00:00'
        >>> parse_timestamp("2021-01-01T12:00:00Z")
        '2021-01-01T12:00:00+00:00'
        >>> parse_timestamp(None)
        None
    """
    if value is None:
        return None

    try:
        # Unix timestamp
        if isinstance(value, (int, float)):
            # Check for infinity and NaN
            import math
            if math.isinf(value) or math.isnan(value):
                return None
            
            dt = datetime.fromtimestamp(value, tz=UTC)
            return dt.isoformat()

        # ISO string
        if isinstance(value, str):
            if not value:  # Empty string
                return None
            # Try parsing as ISO format
            dt = datetime.fromisoformat(value.replace("Z", "+00:00"))
            return dt.isoformat()

        return None
    except (ValueError, OSError, OverflowError):
        return None


def get_current_timestamp() -> str:
    """Get current UTC timestamp in ISO 8601 format.

    Returns:
        ISO 8601 timestamp string

    Examples:
        >>> ts = get_current_timestamp()
        >>> assert ts.endswith('+00:00')
    """
    return datetime.now(UTC).isoformat()


def truncate_string(text: str, max_length: int, suffix: str = "...") -> str:
    """Truncate a string to max_length, adding suffix if truncated.

    Args:
        text: String to truncate
        max_length: Maximum length (including suffix)
        suffix: Suffix to add when truncating (default: "...")

    Returns:
        Truncated string

    Examples:
        >>> truncate_string("Hello World", 8)
        'Hello...'
        >>> truncate_string("Hi", 10)
        'Hi'
    """
    if len(text) <= max_length:
        return text
    
    # If max_length is less than suffix length, just truncate to max_length
    if max_length < len(suffix):
        return text[:max_length]

    return text[: max_length - len(suffix)] + suffix


def safe_dict_get(data: dict[str, Any], *keys: str, default: Any = None) -> Any:
    """Safely get a nested dictionary value.

    Args:
        data: Dictionary to search
        *keys: Path of keys to follow
        default: Default value if path not found

    Returns:
        Value at the path, or default if not found

    Examples:
        >>> data = {"a": {"b": {"c": 42}}}
        >>> safe_dict_get(data, "a", "b", "c")
        42
        >>> safe_dict_get(data, "x", "y", default="missing")
        'missing'
    """
    current = data
    for key in keys:
        if not isinstance(current, dict) or key not in current:
            return default
        current = current[key]
    return current
